package crypt

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/ProtonMail/gopenpgp/v2/helper"
)

const (
	pgpBlockTypeMessage = "PGP MESSAGE"
)

func getEntity(key []byte) (*openpgp.Entity, error) {
	entities, err := openpgp.ReadArmoredKeyRing(bytes.NewBuffer(key))
	if err != nil {
		if strings.Contains(err.Error(), "openpgp: invalid argument: no armored data found") {
			entities, err = openpgp.ReadKeyRing(bytes.NewBuffer(key))
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	if len(entities) <= 0 {
		return nil, errors.New("no entities found in keyring")
	}

	return entities[0], nil
}

func isEntityLocked(entity *openpgp.Entity) (bool, error) {
	if entity.PrivateKey != nil && entity.PrivateKey.Encrypted {
		return true, nil
	}

	return false, nil
}

func IsKeyLocked(key []byte) (bool, string, error) {
	entity, err := getEntity(key)
	if err != nil {
		return false, "", err
	}

	locked, err := isEntityLocked(entity)
	if err != nil {
		return false, "", err
	}

	return locked, hex.EncodeToString(entity.PrimaryKey.Fingerprint), err
}

func Unarmor(data []byte) ([]byte, error) {
	if c, err := armor.Decode(bytes.NewBuffer(data)); err == nil {
		return ioutil.ReadAll(c.Body)
	}

	return data, nil
}

func ReadKey(key []byte, password string) (*openpgp.Entity, string, error) {
	entity, err := getEntity(key)
	if err != nil {
		return nil, "", err
	}

	locked, err := isEntityLocked(entity)
	if err != nil {
		return nil, "", err
	}

	if locked {
		if err := entity.PrivateKey.Decrypt([]byte(password)); err != nil {
			return nil, "", err
		}
		for _, subkey := range entity.Subkeys {
			if err := subkey.PrivateKey.Decrypt([]byte(password)); err != nil {
				return nil, "", err
			}
		}
	}

	return entity, hex.EncodeToString(entity.PrimaryKey.Fingerprint), nil
}

func GenerateKey(
	fullName string,
	email string,
	password string,
) ([]byte, error) { // key, error
	key, err := helper.GenerateKey(fullName, email, []byte(password), "x25519", 0)
	if err != nil {
		return []byte{}, err
	}

	return []byte(key), err
}

type EncryptConfig struct {
	PublicKey       *openpgp.Entity
	ArmorCyphertext bool
}

type SignatureConfig struct {
	PrivateKey      *openpgp.Entity
	ArmorSignature  bool
	DetachSignature bool
}

func EncryptSign(
	encryptConfig *EncryptConfig, // May also be nil
	signatureConfig *SignatureConfig, // May also be nil

	plaintext []byte,
) ([]byte, []byte, error) { // cyphertext, signature, error
	cyphertext := []byte{}
	signature := []byte{}

	if signatureConfig != nil {
		// Sign the plaintext
		buf := &bytes.Buffer{}

		if signatureConfig.DetachSignature {
			if err := openpgp.DetachSign(buf, signatureConfig.PrivateKey, bytes.NewBuffer(plaintext), nil); err != nil {
				return []byte{}, []byte{}, err
			}
		} else {
			w, err := openpgp.Sign(buf, signatureConfig.PrivateKey, nil, nil)
			if err != nil {
				return []byte{}, []byte{}, err
			}

			if _, err := w.Write(plaintext); err != nil {
				return []byte{}, []byte{}, err
			}

			// We have to close before returning, as this adds the footer!
			if err := w.Close(); err != nil {
				return []byte{}, []byte{}, err
			}
		}

		rawSignature, err := ioutil.ReadAll(buf)
		if err != nil {
			return []byte{}, []byte{}, err
		}

		if signatureConfig.ArmorSignature {
			// Armor the signature
			buf := &bytes.Buffer{}

			w, err := armor.Encode(
				buf,
				func() string {
					if signatureConfig.DetachSignature {
						return openpgp.SignatureType
					}

					return pgpBlockTypeMessage
				}(),
				nil,
			)
			if err != nil {
				return []byte{}, []byte{}, err
			}

			if _, err := w.Write(rawSignature); err != nil {
				return []byte{}, []byte{}, err
			}

			// We have to close before returning, as this adds the footer!
			if err := w.Close(); err != nil {
				return []byte{}, []byte{}, err
			}

			armoredSignature, err := ioutil.ReadAll(buf)
			if err != nil {
				return []byte{}, []byte{}, err
			}

			signature = armoredSignature
		} else {
			signature = rawSignature
		}
	}

	if encryptConfig != nil {
		// Encrypt the plaintext
		buf := &bytes.Buffer{}

		w, err := openpgp.Encrypt(buf, []*openpgp.Entity{encryptConfig.PublicKey}, nil, nil, nil)
		if err != nil {
			return []byte{}, []byte{}, err
		}

		if signatureConfig != nil && !signatureConfig.DetachSignature {
			// Encrypt the signature
			if _, err := w.Write(signature); err != nil {
				return []byte{}, []byte{}, err
			}

			signature = []byte{} // Remove signature from return as it is not detached
		} else {
			// Encrypt the plaingtext
			if _, err := w.Write(plaintext); err != nil {
				return []byte{}, []byte{}, err
			}
		}

		// We have to close before returning, as this adds the footer!
		if err := w.Close(); err != nil {
			return []byte{}, []byte{}, err
		}

		rawCyphertext, err := ioutil.ReadAll(buf)
		if err != nil {
			return []byte{}, []byte{}, err
		}

		if encryptConfig.ArmorCyphertext {
			// Armor the cyphertext
			buf := &bytes.Buffer{}

			w, err := armor.Encode(buf, pgpBlockTypeMessage, nil)
			if err != nil {
				return []byte{}, []byte{}, err
			}

			if _, err := w.Write(rawCyphertext); err != nil {
				return []byte{}, []byte{}, err
			}

			// We have to close before returning, as this adds the footer!
			if err := w.Close(); err != nil {
				return []byte{}, []byte{}, err
			}

			armoredCyphertext, err := ioutil.ReadAll(buf)
			if err != nil {
				return []byte{}, []byte{}, err
			}

			cyphertext = armoredCyphertext
		} else {
			cyphertext = rawCyphertext
		}
	}

	return cyphertext, signature, nil
}

type DecryptConfig struct {
	PrivateKey *openpgp.Entity
}

type VerifyConfig struct {
	PublicKey         *openpgp.Entity
	DetachedSignature []byte // May also be armored
}

func DecryptVerify(
	decryptConfig *DecryptConfig, // May also be nil
	verifyConfig *VerifyConfig, // May also be nil

	cyphertext []byte, // May also be armored
) ([]byte, bool, error) { // plaintext, verified, error
	// Unarmor the cyphertext
	text, err := Unarmor(cyphertext)
	if err != nil {
		return []byte{}, false, err
	}

	if decryptConfig != nil {
		// Decrypt the cyphertext
		rawText, err := openpgp.ReadMessage(bytes.NewBuffer(text), openpgp.EntityList{decryptConfig.PrivateKey}, nil, nil)
		if err != nil {
			return []byte{}, false, err
		}

		text, err = ioutil.ReadAll(rawText.UnverifiedBody)
		if err != nil {
			return []byte{}, false, err
		}
	}

	if verifyConfig != nil {
		if verifyConfig.DetachedSignature == nil {
			// Unarmor the signature
			signature, err := Unarmor(text)
			if err != nil {
				return []byte{}, false, err
			}

			// Verify the non-detached signature
			rawText, err := openpgp.ReadMessage(bytes.NewBuffer(signature), openpgp.EntityList{verifyConfig.PublicKey}, nil, nil)
			if err != nil {
				return []byte{}, false, err
			}

			body, err := ioutil.ReadAll(rawText.UnverifiedBody)
			if err != nil {
				return []byte{}, false, err
			}

			hash := rawText.Signature.Hash.New()
			if _, err := hash.Write(body); err != nil {
				return []byte{}, false, err
			}

			if err := verifyConfig.PublicKey.PrimaryKey.VerifySignature(hash, rawText.Signature); err != nil {
				return []byte{}, false, err
			}

			return body, true, nil
		}

		// Verify the detached signature
		rawSignature, err := Unarmor(verifyConfig.DetachedSignature)
		if err != nil {
			return []byte{}, false, err
		}

		r := packet.NewReader(bytes.NewBuffer(rawSignature))
		parsedSignature, err := r.Next()
		if err != nil {
			return []byte{}, false, err
		}

		signature, ok := parsedSignature.(*packet.Signature)
		if !ok {
			return []byte{}, false, errors.New("could not parse signature")
		}

		hash := signature.Hash.New()
		if _, err := hash.Write(text); err != nil {
			return []byte{}, false, err
		}

		if err := verifyConfig.PublicKey.PrimaryKey.VerifySignature(hash, signature); err != nil {
			return []byte{}, false, err
		}

		return text, true, nil
	}

	return text, false, nil
}
