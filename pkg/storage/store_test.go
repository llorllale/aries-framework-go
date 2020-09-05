/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package storage_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/aries-framework-go/pkg/storage"
)

type Provider struct {
	storage.Provider
	Name string
}

func TestStore(t *testing.T) {
	providers := setUpProviders(t)

	for i := range providers {
		provider := providers[i]

		t.Run("Store put and get "+provider.Name, func(t *testing.T) {
			t.Parallel()

			store, err := provider.OpenStore("test")
			require.NoError(t, err)

			const key = "did:example:123"
			data := []byte("value")

			err = store.Put(key, data)
			require.NoError(t, err)

			doc, err := store.Get(key)
			require.NoError(t, err)
			require.NotEmpty(t, doc)
			require.Equal(t, data, doc)

			// test update
			data = []byte(`{"key1":"value1"}`)
			err = store.Put(key, data)
			require.NoError(t, err)

			doc, err = store.Get(key)
			require.NoError(t, err)
			require.NotEmpty(t, doc)
			require.Equal(t, data, doc)

			// test update
			update := []byte(`{"_key1":"value1"}`)
			err = store.Put(key, update)
			require.NoError(t, err)

			doc, err = store.Get(key)
			require.NoError(t, err)
			require.NotEmpty(t, doc)
			require.Equal(t, update, doc)

			did2 := "did:example:789"
			_, err = store.Get(did2)
			require.True(t, errors.Is(err, storage.ErrDataNotFound))

			// nil key
			_, err = store.Get("")
			require.Error(t, err)

			// nil value
			err = store.Put(key, nil)
			require.Error(t, err)

			// nil key
			err = store.Put("", data)
			require.Error(t, err)
		})

		t.Run(fmt.Sprintf("[%s] can open store with uppercase letters in name", provider.Name), func(t *testing.T) {
			name := "UPPERCASE"
			s, err := provider.OpenStore(name)
			require.NoError(t, err)
			require.NotNil(t, s)
			err = provider.CloseStore(name)
			require.NoError(t, err)
		})

		t.Run(fmt.Sprintf("[%s] can put and get keys prefixed with underscore", provider.Name), func(t *testing.T) {
			s, err := provider.OpenStore(fmt.Sprintf("test_%s", randomString()))
			require.NoError(t, err)
			key := "_test_key"
			expected := []byte(uuid.New().String())
			err = s.Put(key, expected)
			require.NoError(t, err)
			result, err := s.Get(key)
			require.NoError(t, err)
			require.Equal(t, expected, result)
		})

		t.Run(fmt.Sprintf("[%s] can iterate keys prefixed with underscore", provider.Name), func(t *testing.T) {
			s, err := provider.OpenStore(fmt.Sprintf("test_%s", randomString()))
			require.NoError(t, err)

			keyTemplate := "_key_%d"
			expectedKeyValues := make(map[string][]byte)

			for i := 0; i < 20; i++ {
				key := fmt.Sprintf(keyTemplate, i)
				val := []byte(uuid.New().String())
				err = s.Put(key, val)
				require.NoError(t, err)
				expectedKeyValues[key] = val
			}

			iter := s.Iterator("_key", fmt.Sprintf("_key%s", storage.EndKeySuffix))
			require.NotNil(t, iter)

			defer iter.Release()

			receivedCount := 0

			for iter.Next() {
				expected, found := expectedKeyValues[string(iter.Key())]
				require.Truef(t, found, "unexpected key %s", iter.Key())
				require.Equal(t, expected, iter.Value())
				receivedCount++
			}

			require.Equal(t, len(expectedKeyValues), receivedCount)
		})

		t.Run(fmt.Sprintf("[%s] can delete keys prefixed with underscore", provider.Name), func(t *testing.T) {
			s, err := provider.OpenStore(fmt.Sprintf("test_%s", randomString()))
			require.NoError(t, err)

			key := "_test_key"

			err = s.Put(key, []byte(uuid.New().String()))
			require.NoError(t, err)

			err = s.Delete(key)
			require.NoError(t, err)

			_, err = s.Get(key)
			require.True(t, errors.Is(err, storage.ErrDataNotFound))
		})

		t.Run("Multi store put and get "+provider.Name, func(t *testing.T) {
			t.Parallel()

			commonKey := uuid.New().String()
			data := []byte("value1")
			// create store 1 & store 2
			store1, err := provider.OpenStore("store-1")
			require.NoError(t, err)

			store2, err := provider.OpenStore("store-2")
			require.NoError(t, err)

			// put in store 1
			err = store1.Put(commonKey, data)
			require.NoError(t, err)

			// get in store 1 - found
			doc, err := store1.Get(commonKey)
			require.NoError(t, err)
			require.NotEmpty(t, doc)
			require.Equal(t, data, doc)

			// get in store 2 - not found
			doc, err = store2.Get(commonKey)
			require.Error(t, err)
			require.True(t, errors.Is(err, storage.ErrDataNotFound))
			require.Empty(t, doc)

			// put in store 2
			err = store2.Put(commonKey, data)
			require.NoError(t, err)

			// get in store 2 - found
			doc, err = store2.Get(commonKey)
			require.NoError(t, err)
			require.NotEmpty(t, doc)
			require.Equal(t, data, doc)

			// create new store 3 with same name as store1
			store3, err := provider.OpenStore("store-1")
			require.NoError(t, err)

			// get in store 3 - found
			doc, err = store3.Get(commonKey)
			require.NoError(t, err)
			require.NotEmpty(t, doc)
			require.Equal(t, data, doc)
		})

		t.Run("Iterator "+provider.Name, func(t *testing.T) {
			t.Parallel()

			store, err := provider.OpenStore("test-iterator")
			require.NoError(t, err)

			const valPrefix = "val-for-%s"
			keys := []string{"abc_123", "abc_124", "abc_125", "abc_126", "jkl_123", "mno_123", "dab_123"}

			for _, key := range keys {
				err = store.Put(key, []byte(fmt.Sprintf(valPrefix, key)))
				require.NoError(t, err)
			}

			itr := store.Iterator("abc_", "abc_"+storage.EndKeySuffix)
			verifyItr(t, itr, 4, "abc_")

			itr = store.Iterator("", "dab_123")
			verifyItr(t, itr, 4, "")

			itr = store.Iterator("abc_124", "")
			verifyItr(t, itr, 0, "")

			itr = store.Iterator("", "")
			verifyItr(t, itr, 0, "")

			itr = store.Iterator("abc_", "mno_"+storage.EndKeySuffix)
			verifyItr(t, itr, 7, "")

			itr = store.Iterator("abc_", "mno_123")
			verifyItr(t, itr, 6, "")
		})

		t.Run("Delete "+provider.Name, func(t *testing.T) {
			t.Parallel()

			const commonKey = "did:example:1234"

			data := []byte("value1")

			// create store 1 & store 2
			store, err := provider.OpenStore("test-store")
			require.NoError(t, err)

			// put in store 1
			err = store.Put(commonKey, data)
			require.NoError(t, err)

			// get in store 1 - found
			doc, err := store.Get(commonKey)
			require.NoError(t, err)
			require.NotEmpty(t, doc)
			require.Equal(t, data, doc)

			// now try Delete with an empty key - should fail
			err = store.Delete("")
			require.EqualError(t, err, "key is mandatory")

			err = store.Delete("k1")
			require.NoError(t, err)

			// finally test Delete an existing key
			err = store.Delete(commonKey)
			require.NoError(t, err)

			doc, err = store.Get(commonKey)
			require.EqualError(t, err, storage.ErrDataNotFound.Error())
			require.Empty(t, doc)
		})
	}
}

func verifyItr(t *testing.T, itr storage.StoreIterator, count int, prefix string) {
	t.Helper()

	var values []string

	for itr.Next() {
		if prefix != "" {
			require.True(t, strings.HasPrefix(string(itr.Key()), prefix))
		}

		values = append(values, string(itr.Value()))
	}
	require.Len(t, values, count)

	itr.Release()
	require.False(t, itr.Next())
	require.Empty(t, itr.Key())
	require.Empty(t, itr.Value())
	require.Error(t, itr.Error())
}

func randomString() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}
