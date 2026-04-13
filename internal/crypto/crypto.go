package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"github.com/mr-tron/base58/base58"
	"github.com/hesusruiz/utils/errl"
)

// DeriveDidKeyFromPrivateKey derives a did:key from an ECDSA private key (P-256).
func DeriveDidKeyFromPrivateKey(privateKey *ecdsa.PrivateKey) (string, error) {
	curve := elliptic.P256()
	uncompressed := elliptic.Marshal(curve, privateKey.PublicKey.X, privateKey.PublicKey.Y)
	if len(uncompressed) != 65 {
		return "", errl.Errorf("unexpected public key length: %d", len(uncompressed))
	}

	xBytes := uncompressed[1:33]
	yLastByte := uncompressed[64]
	var compressedPrefix byte = 0x02
	if yLastByte%2 != 0 {
		compressedPrefix = 0x03
	}

	compressedBytes := append([]byte{compressedPrefix}, xBytes...)
	varintPrefix := []byte{0x80, 0x24} // Varint for P-256
	return "did:key:z" + base58.Encode(append(varintPrefix, compressedBytes...)), nil
}
