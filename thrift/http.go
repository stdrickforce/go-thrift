// Copyright 2012-2015 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package thrift

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/rpc"
)

type http_client_codec struct {
	mb              *memory_buffer
	uri             string
	builder         ProtocolBuilder
	messages        chan message
	current_message message
}

type message struct {
	name  string
	mtype byte
	seq   int32
	err   error
	buf   *memory_buffer
}

func NewHttpClient(p ProtocolBuilder, uri string) *rpc.Client {
	return rpc.NewClientWithCodec(
		NewHttpClientCodec(p, uri),
	)
}

func NewHttpClientCodec(p ProtocolBuilder, uri string) rpc.ClientCodec {
	var mb = NewMemoryBuffer(p, []byte{})
	c := &http_client_codec{
		mb:       mb,
		uri:      uri,
		builder:  p,
		messages: make(chan message),
	}
	return c
}

func (self *http_client_codec) WriteRequest(
	request *rpc.Request,
	thriftStruct interface{},
) error {

	if err := self.mb.WriteMessageBegin(
		request.ServiceMethod,
		MessageTypeCall,
		int32(request.Seq),
	); err != nil {
		return err
	}

	if err := EncodeStruct(self.mb, thriftStruct); err != nil {
		return err
	}

	if err := self.mb.WriteMessageEnd(); err != nil {
		return err
	}

	var buf = self.mb.GetBuffer()

	resp, err := http.Post(self.uri, "application/x-thrift", buf)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New(resp.Status)
	}

	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var mb = NewMemoryBuffer(self.builder, bs)
	name, messageType, seq, err := mb.ReadMessageBegin()
	if err != nil {
		return err
	}

	self.messages <- message{
		name:  name,
		mtype: messageType,
		seq:   seq,
		err:   err,
		buf:   mb,
	}
	return nil
}

func (self *http_client_codec) ReadResponseHeader(response *rpc.Response) error {
	message := <-self.messages

	if message.err != nil {
		return message.err
	}

	response.ServiceMethod = message.name
	response.Seq = uint64(message.seq)

	if message.mtype == MessageTypeException {
		exception := &ApplicationException{}
		if err := DecodeStruct(message.buf, exception); err != nil {
			return err
		}
		response.Error = exception.String()
		return message.buf.ReadMessageEnd()
	}
	self.current_message = message
	return nil
}

func (self *http_client_codec) ReadResponseBody(thriftStruct interface{}) error {
	if thriftStruct == nil {
		return nil
	}

	if err := DecodeStruct(self.current_message.buf, thriftStruct); err != nil {
		return err
	}
	return self.current_message.buf.ReadMessageEnd()
}

func (self *http_client_codec) Close() error {
	// NOTE clean memory buffer.
	return self.mb.Flush()
}
