package urlsigner

import (
	"fmt"
	"strings"
	"time"

	goalone "github.com/bwmarrin/go-alone"
)

type Signer struct {
	Secret []byte
}

func (s *Signer) GenerateTokenFromString(data string) string {
	var urlToSign string
	cryp := goalone.New(s.Secret, goalone.Timestamp)
	if strings.Contains(data, "?") {
		urlToSign = fmt.Sprintf("%s&hash=", data)
	} else {
		urlToSign = fmt.Sprintf("%s?hash=", data)
	}
	tokenByte := cryp.Sign([]byte(urlToSign))
	token := string(tokenByte)
	return token
}

func (s *Signer) VerifyToken(token string) bool {
	cryp := goalone.New(s.Secret, goalone.Timestamp)
	_, err := cryp.Unsign([]byte(token))
	return err == nil
}

func (s *Signer) Expired(token string, minutesUntilExpire int) bool {
	cryp := goalone.New(s.Secret, goalone.Timestamp)
	ts := cryp.Parse([]byte(token))
	return time.Since(ts.Timestamp) > time.Duration(minutesUntilExpire)*time.Minute
}
