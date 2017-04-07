// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build zos

package x509

import "io/ioutil"

// TODO(mundaym): not sure what we need on z/OS for this. Possibly gskssl?
// System SSL docs: http://publibfp.dhe.ibm.com/epubs/pdf/gska1a91.pdf

// Possible certificate files; stop after finding one.
var certFiles = []string{}

func (c *Certificate) systemVerify(opts *VerifyOptions) (chains [][]*Certificate, err error) {
	return nil, nil
}

func initSystemRoots() {
	roots := NewCertPool()
	for _, file := range certFiles {
		data, err := ioutil.ReadFile(file)
		if err == nil {
			roots.AppendCertsFromPEM(data)
			systemRoots = roots
			return
		}
	}

	// All of the files failed to load. systemRoots will be nil which will
	// trigger a specific error at verification time.
}
