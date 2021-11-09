package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"io"
	"log"
)

func main() {
	key := []byte("passphrasewhichneedstobe32bytes!")
	//image, err := crane.Pull("enc")
	ref, err := name.ParseReference("enc")
	if err != nil {
		panic(err)
	}

	image, err := daemon.Image(ref)
	if err != nil {
		panic(err)
	}
	l, err := image.Layers()
	if err != nil {
		panic(err)
	}
	layer1, err := l[0].Compressed()
	if err != nil {
		panic(err)
	}
	//a, err := os.Create("abc.tar.gz")
	//io.Copy(a, layer1)
	//a.Close()

	gzRead, err := gzip.NewReader(layer1)
	if err != nil {
		panic(err)
	}
	insideTar := tar.NewReader(gzRead)
	insideTar.Next()
	body, err := io.ReadAll(insideTar)

	c, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		panic(err)
	}

	nonceSize := gcm.NonceSize()
	if len(body) < nonceSize {
		panic(err)
	}

	nonce, ciphertext := body[:nonceSize], body[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		panic(err)
	}
	tagg, err := name.NewTag("redis:alpine")
	ima, err := tarball.Image(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(plaintext)), nil
	}, &tagg)
	if s, err := daemon.Write(tagg, ima); err != nil {
		log.Fatal(err)
	} else {
		fmt.Println(s)
	}
}
