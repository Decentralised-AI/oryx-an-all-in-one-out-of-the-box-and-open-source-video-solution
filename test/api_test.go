package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	cr "crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ossrs/go-oryx-lib/errors"
	"github.com/ossrs/go-oryx-lib/logger"
)

func TestApi_SetupWebsiteFooter(t *testing.T) {
	ctx, cancel := context.WithTimeout(logger.WithContext(context.Background()), time.Duration(*srsTimeout)*time.Millisecond)
	defer cancel()

	var r0 error
	defer func(ctx context.Context) {
		if err := filterTestError(ctx.Err(), r0); err != nil {
			t.Errorf("Fail for err %+v", err)
		} else {
			logger.Tf(ctx, "test done")
		}
	}(ctx)

	req := struct {
		Beian string `json:"beian"`
		Text  string `json:"text"`
	}{
		Beian: "icp", Text: "TestFooter",
	}
	if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/beian/update", &req, nil); err != nil {
		r0 = err
		return
	}

	res := struct {
		ICP string `json:"icp"`
	}{}
	if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/beian/query", nil, &res); err != nil {
		r0 = err
	} else if res.ICP != "TestFooter" {
		r0 = errors.Errorf("invalid response %v", res)
	}
}

func TestApi_SetupWebsiteTitle(t *testing.T) {
	ctx, cancel := context.WithTimeout(logger.WithContext(context.Background()), time.Duration(*srsTimeout)*time.Millisecond)
	defer cancel()

	var r0 error
	defer func(ctx context.Context) {
		if err := filterTestError(ctx.Err(), r0); err != nil {
			t.Errorf("Fail for err %+v", err)
		} else {
			logger.Tf(ctx, "test done")
		}
	}(ctx)

	var title string
	if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/beian/query", nil, &struct {
		Title *string `json:"title"`
	}{
		Title: &title,
	}); err != nil {
		r0 = err
		return
	} else if title == "" {
		title = "SRS"
	}
	defer func() {
		if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/beian/update", &struct {
			Beian string `json:"beian"`
			Text  string `json:"text"`
		}{
			Beian: "title", Text: title,
		}, nil); err != nil {
			r0 = err
		}
	}()

	req := struct {
		Beian string `json:"beian"`
		Text  string `json:"text"`
	}{
		Beian: "title", Text: "TestTitle",
	}
	if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/beian/update", &req, nil); err != nil {
		r0 = err
		return
	}

	res := struct {
		Title string `json:"title"`
	}{}
	if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/beian/query", nil, &res); err != nil {
		r0 = err
	} else if res.Title != "TestTitle" {
		r0 = errors.Errorf("invalid response %v", res)
	}
}

// Never run this in parallel, because it changes the publish
// secret which might cause other cases to fail.
func TestApi_UpdatePublishSecret(t *testing.T) {
	ctx, cancel := context.WithTimeout(logger.WithContext(context.Background()), time.Duration(*srsTimeout)*time.Millisecond)
	defer cancel()

	var r0, r1 error
	defer func(ctx context.Context) {
		if err := filterTestError(ctx.Err(), r0, r1); err != nil {
			t.Errorf("Fail for err %+v", err)
		} else {
			logger.Tf(ctx, "test done")
		}
	}(ctx)

	var pubSecret string
	if err := NewApi().WithAuth(ctx, "/terraform/v1/hooks/srs/secret/query", nil, &struct {
		Publish *string `json:"publish"`
	}{
		Publish: &pubSecret,
	}); err != nil {
		r0 = err
		return
	} else if pubSecret == "" {
		r0 = errors.Errorf("invalid response %v", pubSecret)
		return
	}

	// Reset the publish secret to the original value.
	defer func() {
		logger.Tf(ctx, "Reset publish secret to %v", pubSecret)
		if err := NewApi().WithAuth(ctx, "/terraform/v1/hooks/srs/secret/update", &struct {
			Secret string `json:"secret"`
		}{
			Secret: pubSecret,
		}, nil); err != nil {
			r1 = err
		}
	}()

	if err := NewApi().WithAuth(ctx, "/terraform/v1/hooks/srs/secret/update", &struct {
		Secret string `json:"secret"`
	}{
		Secret: "TestPublish",
	}, nil); err != nil {
		r0 = err
		return
	}

	res := struct {
		Publish string `json:"publish"`
	}{}
	if err := NewApi().WithAuth(ctx, "/terraform/v1/hooks/srs/secret/query", nil, &res); err != nil {
		r0 = err
	} else if res.Publish != "TestPublish" {
		r0 = errors.Errorf("invalid response %v", res)
	}
}

func TestApi_TutorialsQueryBilibili(t *testing.T) {
	ctx, cancel := context.WithTimeout(logger.WithContext(context.Background()), time.Duration(*srsTimeout)*time.Millisecond)
	defer cancel()

	// If we are using letsencrypt, we don't need to test this.
	if *domainLetsEncrypt != "" || *httpsInsecureVerify {
		return
	}

	if *noBilibiliTest {
		return
	}

	var r0 error
	defer func(ctx context.Context) {
		if err := filterTestError(ctx.Err(), r0); err != nil {
			t.Errorf("Fail for err %+v", err)
		} else {
			logger.Tf(ctx, "test done")
		}
	}(ctx)

	req := struct {
		BVID string `json:"bvid"`
	}{
		BVID: "BV1844y1L7dL",
	}
	res := struct {
		Title string `json:"title"`
		Desc  string `json:"desc"`
	}{}
	if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/bilibili", &req, &res); err != nil {
		r0 = err
	} else if res.Title == "" || res.Desc == "" {
		r0 = errors.Errorf("invalid response %v", res)
	}
}

func TestApi_SslUpdateCert(t *testing.T) {
	ctx, cancel := context.WithTimeout(logger.WithContext(context.Background()), time.Duration(*srsTimeout)*time.Millisecond)
	defer cancel()

	// If we are using letsencrypt, we don't need to test this.
	if *domainLetsEncrypt != "" || *httpsInsecureVerify {
		return
	}

	var r0 error
	defer func(ctx context.Context) {
		if err := filterTestError(ctx.Err(), r0); err != nil {
			t.Errorf("Fail for err %+v", err)
		} else {
			logger.Tf(ctx, "test done")
		}
	}(ctx)

	var key, crt string
	if err := func() error {
		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), cr.Reader)
		if err != nil {
			return errors.Wrapf(err, "generate ecdsa key")
		}

		template := x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject: pkix.Name{
				CommonName: "srs.stack.local",
			},
			NotBefore: time.Now(),
			NotAfter:  time.Now().AddDate(10, 0, 0),
			KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
			ExtKeyUsage: []x509.ExtKeyUsage{
				x509.ExtKeyUsageServerAuth,
			},
			BasicConstraintsValid: true,
		}

		derBytes, err := x509.CreateCertificate(cr.Reader, &template, &template, &privateKey.PublicKey, privateKey)
		if err != nil {
			return errors.Wrapf(err, "create certificate")
		}

		privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
		if err != nil {
			return errors.Wrapf(err, "marshal ecdsa key")
		}

		privateKeyBlock := pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: privateKeyBytes,
		}
		key = string(pem.EncodeToMemory(&privateKeyBlock))

		certBlock := pem.Block{
			Type:  "CERTIFICATE",
			Bytes: derBytes,
		}
		crt = string(pem.EncodeToMemory(&certBlock))
		logger.Tf(ctx, "cert: create self-signed certificate ok, key=%vB, crt=%vB", len(key), len(crt))
		return nil
	}(); err != nil {
		r0 = err
		return
	}

	if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/ssl", &struct {
		Key string `json:"key"`
		Crt string `json:"crt"`
	}{
		Key: key, Crt: crt,
	}, nil); err != nil {
		r0 = err
		return
	}

	conf := struct {
		Provider string `json:"provider"`
		Key      string `json:"key"`
		Crt      string `json:"crt"`
	}{}
	if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/cert/query", nil, &conf); err != nil {
		r0 = err
	} else if conf.Provider != "ssl" || conf.Key != key || conf.Crt != crt {
		r0 = errors.Errorf("invalid response %v", conf)
	}
}

func TestApi_LetsEncryptUpdateCert(t *testing.T) {
	ctx, cancel := context.WithTimeout(logger.WithContext(context.Background()), time.Duration(*srsTimeout)*time.Millisecond)
	defer cancel()

	if *domainLetsEncrypt == "" {
		return
	}

	var r0 error
	defer func(ctx context.Context) {
		if err := filterTestError(ctx.Err(), r0); err != nil {
			t.Errorf("Fail for err %+v", err)
		} else {
			logger.Tf(ctx, "test done")
		}
	}(ctx)

	if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/letsencrypt", &struct {
		Domain string `json:"domain"`
	}{
		Domain: *domainLetsEncrypt,
	}, nil); err != nil {
		r0 = err
		return
	}

	conf := struct {
		Provider string `json:"provider"`
		Key      string `json:"key"`
		Crt      string `json:"crt"`
	}{}
	if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/cert/query", nil, &conf); err != nil {
		r0 = err
	} else if conf.Provider != "lets" || conf.Key == "" || conf.Crt == "" {
		r0 = errors.Errorf("invalid response %v", conf)
	}
}

func TestApi_SetupHpHLSNoHlsCtx(t *testing.T) {
	ctx, cancel := context.WithTimeout(logger.WithContext(context.Background()), time.Duration(*srsTimeout)*time.Millisecond)
	defer cancel()

	var r0 error
	defer func(ctx context.Context) {
		if err := filterTestError(ctx.Err(), r0); err != nil {
			t.Errorf("Fail for err %+v", err)
		} else {
			logger.Tf(ctx, "test done")
		}
	}(ctx)

	type Data struct {
		NoHlsCtx bool `json:"noHlsCtx"`
	}

	if true {
		initData := Data{}
		if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/hphls/query", nil, &initData); err != nil {
			r0 = err
			return
		}
		defer func() {
			if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/hphls/update", &initData, nil); err != nil {
				logger.Tf(ctx, "restore hphls config failed %+v", err)
			}
		}()
	}

	noHlsCtx := Data{NoHlsCtx: true}
	if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/hphls/update", &noHlsCtx, nil); err != nil {
		r0 = err
		return
	}

	verifyData := Data{}
	if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/hphls/query", nil, &verifyData); err != nil {
		r0 = err
		return
	} else if verifyData.NoHlsCtx != true {
		r0 = errors.Errorf("invalid response %+v", verifyData)
	}
}

func TestApi_SetupHpHLSWithHlsCtx(t *testing.T) {
	ctx, cancel := context.WithTimeout(logger.WithContext(context.Background()), time.Duration(*srsTimeout)*time.Millisecond)
	defer cancel()

	var r0 error
	defer func(ctx context.Context) {
		if err := filterTestError(ctx.Err(), r0); err != nil {
			t.Errorf("Fail for err %+v", err)
		} else {
			logger.Tf(ctx, "test done")
		}
	}(ctx)

	type Data struct {
		NoHlsCtx bool `json:"noHlsCtx"`
	}

	if true {
		initData := Data{}
		if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/hphls/query", nil, &initData); err != nil {
			r0 = err
			return
		}
		defer func() {
			if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/hphls/update", &initData, nil); err != nil {
				logger.Tf(ctx, "restore hphls config failed %+v", err)
			}
		}()
	}

	noHlsCtx := Data{NoHlsCtx: false}
	if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/hphls/update", &noHlsCtx, nil); err != nil {
		r0 = err
		return
	}

	verifyData := Data{}
	if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/hphls/query", nil, &verifyData); err != nil {
		r0 = err
		return
	} else if verifyData.NoHlsCtx != false {
		r0 = errors.Errorf("invalid response %+v", verifyData)
	}
}

func TestApi_SetupHlsLowLatencyEnable(t *testing.T) {
	ctx, cancel := context.WithTimeout(logger.WithContext(context.Background()), time.Duration(*srsTimeout)*time.Millisecond)
	defer cancel()

	var r0 error
	defer func(ctx context.Context) {
		if err := filterTestError(ctx.Err(), r0); err != nil {
			t.Errorf("Fail for err %+v", err)
		} else {
			logger.Tf(ctx, "test done")
		}
	}(ctx)

	type Data struct {
		HlsLowLatency bool `json:"hlsLowLatency"`
	}

	if true {
		initData := Data{}
		if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/hlsll/query", nil, &initData); err != nil {
			r0 = err
			return
		}
		defer func() {
			if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/hlsll/update", &initData, nil); err != nil {
				logger.Tf(ctx, "restore hlsll config failed %+v", err)
			}
		}()
	}

	hlsLowLatency := Data{HlsLowLatency: true}
	if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/hlsll/update", &hlsLowLatency, nil); err != nil {
		r0 = err
		return
	}

	verifyData := Data{}
	if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/hlsll/query", nil, &verifyData); err != nil {
		r0 = err
		return
	} else if verifyData.HlsLowLatency != true {
		r0 = errors.Errorf("invalid response %+v", verifyData)
	}
}

func TestApi_SetupHlsLowLatencyDisable(t *testing.T) {
	ctx, cancel := context.WithTimeout(logger.WithContext(context.Background()), time.Duration(*srsTimeout)*time.Millisecond)
	defer cancel()

	var r0 error
	defer func(ctx context.Context) {
		if err := filterTestError(ctx.Err(), r0); err != nil {
			t.Errorf("Fail for err %+v", err)
		} else {
			logger.Tf(ctx, "test done")
		}
	}(ctx)

	type Data struct {
		HlsLowLatency bool `json:"hlsLowLatency"`
	}

	if true {
		initData := Data{}
		if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/hlsll/query", nil, &initData); err != nil {
			r0 = err
			return
		}
		defer func() {
			if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/hlsll/update", &initData, nil); err != nil {
				logger.Tf(ctx, "restore hlsll config failed %+v", err)
			}
		}()
	}

	hlsLowLatency := Data{HlsLowLatency: false}
	if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/hlsll/update", &hlsLowLatency, nil); err != nil {
		r0 = err
		return
	}

	verifyData := Data{}
	if err := NewApi().WithAuth(ctx, "/terraform/v1/mgmt/hlsll/query", nil, &verifyData); err != nil {
		r0 = err
		return
	} else if verifyData.HlsLowLatency != false {
		r0 = errors.Errorf("invalid response %+v", verifyData)
	}
}

func TestApi_SrsApiNoAuth(t *testing.T) {
	ctx, cancel := context.WithTimeout(logger.WithContext(context.Background()), time.Duration(*srsTimeout)*time.Millisecond)
	defer cancel()

	var r0 error
	defer func(ctx context.Context) {
		if err := filterTestError(ctx.Err(), r0); err != nil {
			t.Errorf("Fail for err %+v", err)
		} else {
			logger.Tf(ctx, "test done")
		}
	}(ctx)

	var pubSecret string
	if err := NewApi().WithAuth(ctx, "/terraform/v1/hooks/srs/secret/query", nil, &struct {
		Publish *string `json:"publish"`
	}{
		Publish: &pubSecret,
	}); err != nil {
		r0 = err
		return
	}

	// Should OK for RTC api.
	offer := strings.ReplaceAll(SrsLarixExampleOffer, "\n", "\r\n")
	streamID := fmt.Sprintf("stream-%v-%v", os.Getpid(), rand.Int())
	if err := NewApi().NoAuth(ctx, fmt.Sprintf("/rtc/v1/whip/?app=live&stream=%v&secret=%v", streamID, pubSecret), offer, nil); err != nil {
		r0 = errors.Wrapf(err, "should ok for rtc publish api")
		return
	}

	// For health check api, should ok.
	if err := NewApi().NoAuth(ctx, "/api/v1/versions", nil, nil); err != nil {
		r0 = errors.Wrapf(err, "should ok for health check api")
		return
	}

	// Should failed if no auth.
	if err := NewApi().NoAuth(ctx, "/api/v1/summaries", nil, nil); err == nil {
		r0 = errors.Errorf("should failed if no auth")
		return
	}
}

func TestApi_SrsApiWithAuth(t *testing.T) {
	ctx, cancel := context.WithTimeout(logger.WithContext(context.Background()), time.Duration(*srsTimeout)*time.Millisecond)
	defer cancel()

	var r0 error
	defer func(ctx context.Context) {
		if err := filterTestError(ctx.Err(), r0); err != nil {
			t.Errorf("Fail for err %+v", err)
		} else {
			logger.Tf(ctx, "test done")
		}
	}(ctx)

	var pubSecret string
	if err := NewApi().WithAuth(ctx, "/terraform/v1/hooks/srs/secret/query", nil, &struct {
		Publish *string `json:"publish"`
	}{
		Publish: &pubSecret,
	}); err != nil {
		r0 = err
		return
	}

	ver := struct {
		Major    int    `json:"major"`
		Minor    int    `json:"minor"`
		Revision int    `json:"revision"`
		Version  string `json:"version"`
	}{}
	if err := NewApi().WithAuth(ctx, "/api/v1/versions", nil, &ver); err != nil {
		r0 = errors.Wrapf(err, "request failed")
	} else if ver.Major != 5 {
		r0 = errors.Errorf("invalid response %v", ver)
	}

	summaries := struct {
		OK   bool `json:"ok"`
		Self struct {
			Version string `json:"version"`
		} `json:"self"`
		System struct {
			CPUs int `json:"cpus"`
		} `json:"system"`
	}{}
	if err := NewApi().WithAuth(ctx, "/api/v1/summaries", nil, &summaries); err != nil {
		r0 = errors.Wrapf(err, "request failed")
	} else if ver.Version != summaries.Self.Version {
		r0 = errors.Errorf("invalid response %v %v", summaries, ver)
	} else if summaries.System.CPUs <= 0 {
		r0 = errors.Errorf("invalid response %v", summaries)
	}

	// Should OK for RTC api.
	offer := strings.ReplaceAll(SrsLarixExampleOffer, "\n", "\r\n")
	streamID := fmt.Sprintf("stream-%v-%v", os.Getpid(), rand.Int())
	if err := NewApi().WithAuth(ctx, fmt.Sprintf("/rtc/v1/whip/?app=live&stream=%v&secret=%v", streamID, pubSecret), offer, nil); err != nil {
		r0 = errors.Wrapf(err, "should ok for rtc publish api")
		return
	}
}

func TestApi_SrsApiCorsNoOrigin(t *testing.T) {
	ctx, cancel := context.WithTimeout(logger.WithContext(context.Background()), time.Duration(*srsTimeout)*time.Millisecond)
	defer cancel()

	var r0, r1 error
	defer func(ctx context.Context) {
		if err := filterTestError(ctx.Err(), r0, r1); err != nil {
			t.Errorf("Fail for err %+v", err)
		} else {
			logger.Tf(ctx, "test done")
		}
	}(ctx)

	var pubSecret string
	if err := NewApi().WithAuth(ctx, "/terraform/v1/hooks/srs/secret/query", nil, &struct {
		Publish *string `json:"publish"`
	}{
		Publish: &pubSecret,
	}); err != nil {
		r0 = err
		return
	}

	offer := strings.ReplaceAll(SrsLarixExampleOffer, "\n", "\r\n")
	streamID := fmt.Sprintf("stream-%v-%v", os.Getpid(), rand.Int())

	for _, api := range []struct {
		Api  string
		Data interface{}
	}{
		{"/terraform/v1/host/versions", nil},
		{"/api/v1/versions", nil},
		{fmt.Sprintf("/rtc/v1/whip/?app=live&stream=%v&secret=%v", streamID, pubSecret), offer},
	} {
		if err := NewApi(func(v *testApi) {
			v.InjectResponse = func(resp *http.Response) {
				for _, header := range []struct {
					Key, Value string
				}{
					{"Access-Control-Allow-Origin", "*"},
					{"Access-Control-Allow-Headers", "*"},
					{"Access-Control-Allow-Methods", "*"},
					{"Access-Control-Expose-Headers", "*"},
					{"Access-Control-Allow-Credentials", "true"},
				} {
					if value := resp.Header.Get(header.Key); value != header.Value {
						r1 = errors.Errorf("invalid CORS %v=%v, expect %v", header.Key, header.Value, value)
					}
					if values := resp.Header.Values(header.Key); len(values) != 1 {
						r1 = errors.Errorf("invalid CORS %v=%v, expect only one", header.Key, values)
					}
				}
			}
		}).NoAuth(ctx, api.Api, api.Data, nil); err != nil {
			r0 = errors.Errorf("should be ok for api %v", api)
			return
		}
		if r1 != nil {
			r1 = errors.Wrapf(r1, "should be ok for api %v", api)
			return
		}
	}
}

func TestApi_SrsApiCorsWithOrigin(t *testing.T) {
	ctx, cancel := context.WithTimeout(logger.WithContext(context.Background()), time.Duration(*srsTimeout)*time.Millisecond)
	defer cancel()

	var r0, r1 error
	defer func(ctx context.Context) {
		if err := filterTestError(ctx.Err(), r0, r1); err != nil {
			t.Errorf("Fail for err %+v", err)
		} else {
			logger.Tf(ctx, "test done")
		}
	}(ctx)

	var pubSecret string
	if err := NewApi().WithAuth(ctx, "/terraform/v1/hooks/srs/secret/query", nil, &struct {
		Publish *string `json:"publish"`
	}{
		Publish: &pubSecret,
	}); err != nil {
		r0 = err
		return
	}

	offer := strings.ReplaceAll(SrsLarixExampleOffer, "\n", "\r\n")
	streamID := fmt.Sprintf("stream-%v-%v", os.Getpid(), rand.Int())

	for _, api := range []struct {
		Api  string
		Data interface{}
	}{
		{"/terraform/v1/host/versions", nil},
		{"/api/v1/versions", nil},
		{fmt.Sprintf("/rtc/v1/whip/?app=live&stream=%v&secret=%v", streamID, pubSecret), offer},
	} {
		if err := NewApi(func(v *testApi) {
			v.InjectRequest = func(req *http.Request) {
				req.Header.Set("Origin", "http://always-cors.ossrs.io")
			}
			v.InjectResponse = func(resp *http.Response) {
				for _, header := range []struct {
					Key, Value string
				}{
					{"Access-Control-Allow-Origin", "*"},
					{"Access-Control-Allow-Headers", "*"},
					{"Access-Control-Allow-Methods", "*"},
					{"Access-Control-Expose-Headers", "*"},
					{"Access-Control-Allow-Credentials", "true"},
				} {
					if value := resp.Header.Get(header.Key); value != header.Value {
						r1 = errors.Errorf("invalid CORS %v=%v, expect %v", header.Key, header.Value, value)
					}
					if values := resp.Header.Values(header.Key); len(values) != 1 {
						r1 = errors.Errorf("invalid CORS %v=%v, expect only one", header.Key, values)
					}
				}
			}
		}).NoAuth(ctx, api.Api, api.Data, nil); err != nil {
			r0 = errors.Errorf("should be ok for api %v", api)
			return
		}
		if r1 != nil {
			r1 = errors.Wrapf(r1, "should be ok for api %v", api)
			return
		}
	}
}
