// Copyright 2015 The kube-cluster Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 		http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package virtualbox

import (
	"archive/tar"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
)

// todo(carlos): detect CPU architecture and download appropriate ISO
// todo(carlos): detect OS and adjust VBoxManageBin properly (.exe)
// todo(carlos): test for minimum VBox version

const (
	VagrantBox = "https://cloud-images.ubuntu.com/vagrant/vivid/current/vivid-server-cloudimg-amd64-vagrant-disk1.box"
)

func (v *Virtualbox) Setup() error {
	// todo(carlos): detect CPU architecture and adjust ostype accordingly

	steps := []func() error{
		v.downloadISO,
		v.untarBox,
		v.importVM,
		v.startVM,
	}

	for _, step := range steps {
		fmt.Print(".")
		if err := step(); err != nil {
			return err
		}
	}

	return nil
}

func (v *Virtualbox) downloadISO() error {
	// todo(carlos): if the ISO is available local, and it is consistent, then avoid the second download.
	output, err := os.Create(path.Base(VagrantBox))
	if err != nil {
		return err
	}
	defer output.Close()

	response, err := http.Get(VagrantBox)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if _, err := io.Copy(output, response.Body); err != nil {
		return err
	}

	return nil
}

func (v *Virtualbox) untarBox() error {
	boxFileReader, err := os.Open(path.Base(VagrantBox))
	if err != nil {
		return err
	}
	defer boxFileReader.Close()

	tarReader := tar.NewReader(boxFileReader)
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		fn := hdr.Name
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(fn, os.FileMode(hdr.Mode)); err != nil {
				return err
			}

		case tar.TypeReg:
			writer, err := os.Create(fn)
			if err != nil {
				return err
			}
			io.Copy(writer, tarReader)
			if err = os.Chmod(fn, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
			writer.Close()

		default:
			log.Panicf("unable to untar type: %c in file %s", hdr.Typeflag, fn)
		}
	}

	return nil
}

func (v *Virtualbox) importVM() error {
	out, err := exec.Command(v.mgmtbin, "import", "box.ovf", "--vsys", "0", "--vmname", v.envName).CombinedOutput()
	if err != nil {
		log.Printf("%s\n", out)
		return err
	}
	return nil
}

func (v *Virtualbox) startVM() error {
	if err := exec.Command(v.headlessbin, "-s", v.envName).Start(); err != nil {
		return err
	}
	return nil
}
