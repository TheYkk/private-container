package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"

	"io"
	"log"
)

func main() {
	key := []byte("passphrasewhichneedstobe32bytes!")
	ref, err := name.ParseReference("redis:alpine")
	if err != nil {
		panic(err)
	}

	image, err := daemon.Image(ref)
	if err != nil {
		panic(err)
	}

	buf := &bytes.Buffer{}
	waiter := make(chan bool)
	go func() {
		err := tarball.Write(ref, image, buf)
		if err != nil {
			panic(err)
		}
		waiter <- true
		if err != nil {
			panic(err)
		}
	}()
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Fatal(err)
	}

	//Create a new GCM - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	//https://golang.org/pkg/crypto/cipher/#NewGCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		log.Fatal(err)
	}

	//Create a nonce. Nonce should be from GCM
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		log.Fatal(err)
	}

	//Encrypt the data using aesGCM.Seal
	//Since we don't want to save the nonce somewhere else in this case, we add it as a prefix to the encrypted data. The first nonce argument in Seal is the prefix.
	<-waiter
	ciphertext := aesGCM.Seal(nonce, nonce, buf.Bytes(), nil)

	tarBuf := &bytes.Buffer{}
	gzipWriter := gzip.NewWriter(tarBuf)
	tarWriter := tar.NewWriter(gzipWriter)

	header := &tar.Header{
		Name: "/encrypted.image",
		Size: int64(len(ciphertext)),
		Mode: 0600,
	}
	err = tarWriter.WriteHeader(header)
	if err != nil {
		log.Fatal(err)
	}

	_, err = tarWriter.Write(ciphertext)
	if err != nil {
		log.Fatal(err)
	}
	tarWriter.Close()
	gzipWriter.Close()

	imgScratch := empty.Image

	tarLayer, err := tarball.LayerFromReader(tarBuf)

	newImg, err := mutate.AppendLayers(imgScratch, tarLayer)
	if err != nil {
		log.Fatal(err)
	}
	tag, err := name.NewTag("enc")
	if err != nil {
		log.Fatal(err)
	}

	if s, err := daemon.Write(tag, newImg); err != nil {
		log.Fatal(err)
	} else {
		fmt.Println(s)
	}

}
