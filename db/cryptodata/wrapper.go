// Copyright (c) 2018 Cisco and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cryptodata

import (
	"github.com/ligato/cn-infra/db/keyval"
	"github.com/golang/protobuf/proto"
)

// decryptData holds values required for decrypting
type decryptData struct {
	// Function used for decrypting arbitrary data later
	decryptFunc DecryptFunc
	// ArbitraryDecrypter is used to decrypt data
	decrypter ArbitraryDecrypter
}

// KvBytesPluginWrapper wraps keyval.KvBytesPlugin with additional support of reading encrypted data
type KvBytesPluginWrapper struct {
	keyval.KvBytesPlugin
	decryptData
}

// BytesBrokerWrapper wraps keyval.BytesBroker with additional support of reading encrypted data
type BytesBrokerWrapper struct {
	keyval.BytesBroker
	decryptData
}

// BytesWatcherWrapper wraps keyval.BytesWatcher with additional support of reading encrypted data
type BytesWatcherWrapper struct {
	keyval.BytesWatcher
	decryptData
}

// BytesKeyValWrapper wraps keyval.BytesKeyVal with additional support of reading encrypted data
type BytesKeyValWrapper struct {
	keyval.BytesKeyVal
	decryptData
}

// BytesWatchRespWrapper wraps keyval.BytesWatchResp with additional support of reading encrypted data
type BytesWatchRespWrapper struct {
	keyval.BytesWatchResp
	BytesKeyValWrapper
}

// BytesKeyValIterator wraps keyval.BytesKeyValIterator with additional support of reading encrypted data
type BytesKeyValIteratorWrapper struct {
	keyval.BytesKeyValIterator
	decryptData
}

// NewKvBytesPluginWrapper creates wrapper for provided CoreBrokerWatcher, adding support for decrypting encrypted
// data
func NewKvBytesPluginWrapper(cbw keyval.KvBytesPlugin, decrypter ArbitraryDecrypter, decryptFunc DecryptFunc) *KvBytesPluginWrapper {
	return &KvBytesPluginWrapper{
		KvBytesPlugin: cbw,
		decryptData: decryptData{
			decryptFunc: decryptFunc,
			decrypter:   decrypter,
		},
	}
}

// NewBytesBrokerWrapper creates wrapper for provided BytesBroker, adding support for decrypting encrypted data
func NewBytesBrokerWrapper(pb keyval.BytesBroker, decrypter ArbitraryDecrypter, decryptFunc DecryptFunc) *BytesBrokerWrapper {
	return &BytesBrokerWrapper{
		BytesBroker: pb,
		decryptData: decryptData{
			decryptFunc: decryptFunc,
			decrypter:   decrypter,
		},
	}
}

// NewBytesWatcherWrapper creates wrapper for provided BytesWatcher, adding support for decrypting encrypted data
func NewBytesWatcherWrapper(pb keyval.BytesWatcher, decrypter ArbitraryDecrypter, decryptFunc DecryptFunc) *BytesWatcherWrapper {
	return &BytesWatcherWrapper{
		BytesWatcher: pb,
		decryptData: decryptData{
			decryptFunc: decryptFunc,
			decrypter:   decrypter,
		},
	}
}

// NewBroker returns a BytesBroker instance with support for decrypting values that prepends given <keyPrefix> to all
// keys in its calls.
// To avoid using a prefix, pass keyval.Root constant as argument.
func (cbw *KvBytesPluginWrapper) NewBroker(prefix string) keyval.BytesBroker {
	return NewBytesBrokerWrapper(cbw.KvBytesPlugin.NewBroker(prefix), cbw.decrypter, cbw.decryptFunc)
}

// NewWatcher returns a BytesWatcher instance with support for decrypting values that prepends given <keyPrefix> to all
// keys during watch subscribe phase.
// The prefix is removed from the key retrieved by GetKey() in BytesWatchResp.
// To avoid using a prefix, pass keyval.Root constant as argument.
func (cbw *KvBytesPluginWrapper) NewWatcher(prefix string) keyval.BytesWatcher {
	return NewBytesWatcherWrapper(cbw.KvBytesPlugin.NewWatcher(prefix), cbw.decrypter, cbw.decryptFunc)
}

// GetValue retrieves and tries to decrypt one item under the provided key.
func (cbb *BytesBrokerWrapper) GetValue(key string) (data []byte, found bool, revision int64, err error) {
	data, found, revision, err = cbb.BytesBroker.GetValue(key)
	if err == nil {
		objData, err := cbb.decrypter.Decrypt(data, cbb.decryptFunc)
		if err != nil {
			return data, found, revision, err
		}

		data = objData.([]byte)
	}
	return
}

// ListValues returns an iterator that enables to traverse all items stored
// under the provided <key>.
func (cbb *BytesBrokerWrapper) ListValues(key string) (keyval.BytesKeyValIterator, error) {
	kv, err := cbb.BytesBroker.ListValues(key)
	if err != nil {
		return kv, err
	}
	return &BytesKeyValIteratorWrapper{
		BytesKeyValIterator: kv,
		decryptData:         cbb.decryptData,
	}, nil
}

// Watch starts subscription for changes associated with the selected keys.
// Watch events will be delivered to callback (not channel) <respChan>.
// Channel <closeChan> can be used to close watching on respective key
func (b *BytesWatcherWrapper) Watch(respChan func(keyval.BytesWatchResp), closeChan chan string, keys ...string) error {
	return b.BytesWatcher.Watch(func(resp keyval.BytesWatchResp) {
		respChan(&BytesWatchRespWrapper{
			BytesWatchResp: resp,
			BytesKeyValWrapper: BytesKeyValWrapper{
				BytesKeyVal: resp,
				decryptData: b.decryptData,
			},
		})
	}, closeChan, keys...)
}

// GetValue returns the value of the pair.
func (r *BytesWatchRespWrapper) GetValue() []byte {
	return r.BytesKeyValWrapper.GetValue()
}

// GetPrevValue returns the previous value of the pair.
func (r *BytesWatchRespWrapper) GetPrevValue() []byte {
	return r.BytesKeyValWrapper.GetPrevValue()
}

// GetValue returns the value of the pair.
func (r *BytesKeyValWrapper) GetValue() []byte {
	data, err := r.decrypter.Decrypt(r.BytesKeyVal.GetValue(), r.decryptFunc)
	if err != nil {
		return nil
	}
	return data.([]byte)
}

// GetPrevValue returns the previous value of the pair.
func (r *BytesKeyValWrapper) GetPrevValue() []byte {
	data, err := r.decrypter.Decrypt(r.BytesKeyVal.GetPrevValue(), r.decryptFunc)
	if err != nil {
		return nil
	}
	return data.([]byte)
}

// GetNext retrieves the following item from the context.
// When there are no more items to get, <stop> is returned as *true*
// and <kv> is simply *nil*.
func (r *BytesKeyValIteratorWrapper) GetNext() (kv keyval.BytesKeyVal, stop bool) {
	if stop {
		return kv, stop
	}
	return &BytesKeyValWrapper{
		BytesKeyVal: kv,
		decryptData: r.decryptData,
	}, stop
}

// KvProtoPluginWrapper wraps keyval.KvProtoPlugin with additional support of reading encrypted data
type KvProtoPluginWrapper struct {
	keyval.KvProtoPlugin
	decryptData
}

// ProtoBrokerWrapper wraps keyval.ProtoBroker with additional support of reading encrypted data
type ProtoBrokerWrapper struct {
	keyval.ProtoBroker
	decryptData
}

// NewKvProtoPluginWrapper creates wrapper for provided KvProtoPlugin, adding support for decrypting encrypted data
func NewKvProtoPluginWrapper(kvp keyval.KvProtoPlugin, decrypter ArbitraryDecrypter, decryptFunc DecryptFunc) *KvProtoPluginWrapper {
	return &KvProtoPluginWrapper{
		KvProtoPlugin: kvp,
		decryptData: decryptData{
			decryptFunc: decryptFunc,
			decrypter:   decrypter,
		},
	}
}

// NewProtoBrokerWrapper creates wrapper for provided ProtoBroker, adding support for decrypting encrypted data
func NewProtoBrokerWrapper(pb keyval.ProtoBroker, decrypter ArbitraryDecrypter, decryptFunc DecryptFunc) *ProtoBrokerWrapper {
	return &ProtoBrokerWrapper{
		ProtoBroker: pb,
		decryptData: decryptData{
			decryptFunc: decryptFunc,
			decrypter:   decrypter,
		},
	}
}

// NewBroker returns a BytesBroker instance with support for decrypting values that prepends given <keyPrefix> to all
// keys in its calls.
// To avoid using a prefix, pass keyval.Root constant as argument.
func (kvp *KvProtoPluginWrapper) NewBroker(prefix string) keyval.ProtoBroker {
	return NewProtoBrokerWrapper(kvp.KvProtoPlugin.NewBroker(prefix), kvp.decrypter, kvp.decryptFunc)
}

// GetValue retrieves one item under the provided <key>. If the item exists,
// it is unmarshaled into the <reqObj> and its fields are decrypted.
func (db *ProtoBrokerWrapper) GetValue(key string, reqObj proto.Message) (bool, int64, error) {
	found, revision, err := db.ProtoBroker.GetValue(key, reqObj)
	if !found || err != nil {
		return found, revision, err
	}

	_, err = db.decrypter.Decrypt(reqObj, db.decryptFunc)
	return found, revision, err
}
