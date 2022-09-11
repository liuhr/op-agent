package oraft

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/openark/golib/log"
	"op-agent/config"
	"op-agent/ssl"
)

var httpClient *http.Client

func setupHttpClient() error {
	httpTimeout := time.Duration(config.ActiveNodeExpireSeconds) * time.Second
	dialTimeout := func(network, addr string) (net.Conn, error) {
		return net.DialTimeout(network, addr, httpTimeout)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.Config.SSLSkipVerify,
	}
	if config.Config.UseSSL {
		caPool, err := ssl.ReadCAFile(config.Config.SSLCAFile)
		if err != nil {
			return err
		}
		tlsConfig.RootCAs = caPool

		if config.Config.UseMutualTLS {
			var sslPEMPassword []byte
			if ssl.IsEncryptedPEM(config.Config.SSLPrivateKeyFile) {
				sslPEMPassword = ssl.GetPEMPassword(config.Config.SSLPrivateKeyFile)
			}
			if err := ssl.AppendKeyPairWithPassword(tlsConfig, config.Config.SSLCertFile, config.Config.SSLPrivateKeyFile, sslPEMPassword); err != nil {
				return err
			}
		}
	}

	httpTransport := &http.Transport{
		TLSClientConfig:       tlsConfig,
		Dial:                  dialTimeout,
		ResponseHeaderTimeout: httpTimeout,
	}
	httpClient = &http.Client{Transport: httpTransport}

	return nil
}

func HttpGetLeader(path string) (response []byte, err error) {
	leaderURI := LeaderURI.Get()
	if leaderURI == "" {
		return nil, fmt.Errorf("Raft leader URI unknown")
	}
	leaderAPI := leaderURI
	if config.Config.URLPrefix != "" {
		// We know URLPrefix begind with "/"
		leaderAPI = fmt.Sprintf("%s%s", leaderAPI, config.Config.URLPrefix)
	}
	leaderAPI = fmt.Sprintf("%s/api", leaderAPI)

	url := fmt.Sprintf("%s/%s", leaderAPI, path)

	req, err := http.NewRequest("GET", url, nil)
	switch strings.ToLower(config.Config.AuthenticationMethod) {
	case "basic", "multi":
		req.SetBasicAuth(config.Config.HTTPAuthUser, config.Config.HTTPAuthPassword)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return body, log.Errorf("HttpGetLeader: got %d status on %s", res.StatusCode, url)
	}

	return body, nil
}
