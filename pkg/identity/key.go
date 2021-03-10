package identity

import (
	"errors"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
)

func NewKeyPair(priKeyFile, pubKeyFile string) error {
	var pubKey crypto.PubKey
	// nolint: gosec
	r := rand.New(rand.NewSource(time.Now().Unix() + 789))
	priKey, pubKey, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return err
	}
	return SaveKeyPair(priKey, pubKey, priKeyFile, pubKeyFile)
}

func SaveKeyPair(priKey crypto.PrivKey, pubKey crypto.PubKey, priKeyFile, pubKeyFile string) error {
	if priKey != nil {
		if priKeyFile == "" {
			return errors.New("no pri key file")
		}
	}
	if pubKey != nil {
		if pubKeyFile == "" {
			return errors.New("no pub key file")
		}
	}

	if priKey != nil {
		d, err := crypto.MarshalPrivateKey(priKey)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(priKeyFile, d, os.ModePerm)
		if err != nil {
			return err
		}
	}

	if pubKey != nil {
		d, err := crypto.MarshalPublicKey(pubKey)
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(pubKeyFile, d, os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

func LoadKeyPair(priKeyFile, pubKeyFile string) (priKey crypto.PrivKey, pubKey crypto.PubKey, err error) {
	var d []byte
	if priKeyFile != "" {
		d, err = ioutil.ReadFile(priKeyFile)
		if err != nil {
			return
		}
		priKey, err = crypto.UnmarshalPrivateKey(d)
		if err != nil {
			return
		}
	}

	if pubKeyFile != "" {
		d, err = ioutil.ReadFile(pubKeyFile)
		if err != nil {
			return
		}

		pubKey, err = crypto.UnmarshalPublicKey(d)
		if err != nil {
			return
		}
	}

	return
}
