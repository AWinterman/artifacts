package artifacts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

func TestContract(t *testing.T) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	clientOfAServiceV1 := ArtifactId{
		Namespace: "client",
		Package:   "of.a.service",
		Version:   "1",
	}
	badArtifactOne := ArtifactId{
		Package: "fuuuu-common",
		Version: "000",
	}
	badArtifactTwo := ArtifactId{
		Namespace: "fuuuu",
		Package:   "co-common",
		Version:   "999",
	}
	artifacts := []Artifact{
		{
			ArtifactId: clientOfAServiceV1,
			Repository: "internal",
			Format:     "maven",
			Error:      nil,
			Status:     Published,
			CreateTime: time.Now(),
		},
		{
			ArtifactId: ArtifactId{
				Namespace: "client",
				Package:   "of.a.service",
				Version:   "0.1.0",
			},
			Repository: "internal",
			Format:     "maven",
			Error:      nil,
			Status:     Unlisted,
			CreateTime: time.Now(),
		},
		{
			ArtifactId: badArtifactTwo,
			Repository: "internal",
			Format:     "maven",
			Status:     Published,
			CreateTime: time.Now(),
			Problems:   []string{"ONE"},
		},
		{
			ArtifactId: badArtifactOne,
			Repository: "internal",
			Format:     "maven",
			Error:      nil,
			Status:     Published,
			CreateTime: time.Now(),
		},
		{
			ArtifactId: ArtifactId{
				Package:   "co-common",
				Version:   "1.1.0",
				Namespace: "HII",
			},
			Repository: "internal",
			Format:     "maven",
			Status:     Published,
			CreateTime: time.Now(),
		},
	}

	t.Run("round trip", func(t *testing.T) {

		server, addr, dbFile := setupAndStartServer(t)
		defer func(server *http.Server) {
			t.Log("Closing server")
			_ = server.Shutdown(context.Background())
			t.Logf("Deleting test db file")
			_ = os.Remove(dbFile)
		}(server)

		t.Run("Inserts successfully", func(t *testing.T) {
			insert := url.URL{
				Scheme: "http",
				Host:   addr,
				Path:   "/",
			}

			j, err := json.Marshal(artifacts)
			request, err := http.NewRequest("PUT", insert.String(), bytes.NewBuffer(j))
			if err != nil {
				t.Fatal(err)
			}
			response, err := http.DefaultClient.Do(request)

			if err != nil {
				t.Fatal(err)
			}
			i, err := ReadResponse(response)
			if err != nil && err != io.EOF {
				t.Fatal(err)
			}

			t.Run("Insert Response value matches expectations", func(t *testing.T) {
				if response.StatusCode < 300 {
					m, err := UnmarshalArtifactList(i)
					if err != io.EOF && err != nil {
						t.Fatalf("got error reading response \n%v \n%v \n%s", response, err, i)
					}
					foundV1ClientOfAService := false
					for _, artifact := range m {
						t.Logf("%+v %v", artifact.ArtifactId, artifact.Problems)
						if len(artifact.Problems) > 0 {
							if artifact.ArtifactId.Version != badArtifactTwo.Version && artifact.ArtifactId.Version != badArtifactOne.Version {
								t.Fatalf("unexpected bad artifact %v\n%s", artifact.ArtifactId, i)
							}
						}
						if clientOfAServiceV1 == artifact.ArtifactId {
							foundV1ClientOfAService = true
						}

					}
					if !foundV1ClientOfAService {
						t.Fatalf("Did not insert expected artifact %+v", clientOfAServiceV1)
					}
				} else {
					t.Fatalf("Request failed %s %s", response.Status, string(i))
				}
			})
		})

		t.Run("Can fetch all data", func(t *testing.T) {

			u := url.URL{
				Scheme: "http",
				Host:   addr,
				Path:   "/",
			}
			newRequest, err := http.NewRequest("GET", u.String(), http.NoBody)
			if err != nil {
				t.Fatal(err)
			}
			newRequest.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(newRequest)

			if resp.StatusCode > 300 {
				t.Fatalf("Failed to make request %+v", resp)
			}
			b, err := ReadResponse(resp)

			if err != nil && err != io.EOF {
				t.Fatalf("Failed making request %+e", err)
			}

			list, err := UnmarshalArtifactList(b)
			if err != nil {
				t.Fatalf("Failed unmarshalling response %+e", err)
			}
			if len(list) != 3 {
				t.Logf("Wrong number of elements in response %#v", len(list))
				t.Fail()
				for i2, i3 := range list {
					t.Log(i2, i3)
				}
			}
		})

		t.Run("Can fetch data with the 'client' package", func(t *testing.T) {

			u := url.URL{
				Scheme:   "http",
				Host:     addr,
				Path:     "/",
				RawQuery: "package=client",
			}

			t.Logf("Request Url %s", u.String())
			newRequest, err := http.NewRequest("GET", u.String(), http.NoBody)
			if err != nil {
				t.Fatal(err)
			}
			newRequest.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(newRequest)
			if resp.StatusCode > 300 {
				t.Fatalf("Failed to make request %+v", resp)
			}
			b, err := ReadResponse(resp)
			if err != nil && err != io.EOF {
				t.Fatalf("Failed making request %+e", err)
			}

			list, err := UnmarshalArtifactList(b)
			if err != nil {
				t.Fatalf("Failed unmarshalling response %+e", err)
			}
			if len(list) != 2 {
				t.Logf("Wrong number of elements in response %#v", len(list))
				t.Fail()
				for i2, i3 := range list {
					t.Log(i2, i3)
				}
			}
		})

	})
}

func UnmarshalArtifactList(i []byte) ([]Artifact, error) {
	var m []Artifact
	err := json.Unmarshal(i, &m)
	return m, err
}

func ReadResponse(response *http.Response) ([]byte, error) {
	length := response.ContentLength
	i := make([]byte, length)
	_, err := response.Body.Read(i)
	return i, err
}

func setupAndStartServer(t *testing.T) (*http.Server, string, string) {
	testFileName := fmt.Sprintf(".test.%s", uuid.New())
	err := os.Setenv("ARTIFACTS_DBFILE", testFileName)
	if err != nil {
		t.Error(err)
	}
	err = os.Setenv("ARTIFACTS_TEMPLATES", "templates")
	if err != nil {
		t.Fatal(err)
	}

	specification, err := LoadSpecification()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	storage, err := NewStorage(specification)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}

	server := NewServer("", storage)
	go func() {
		err := server.Serve(listener)
		if err != http.ErrServerClosed {
			t.Error(err)
			t.Fail()
		}
	}()

	return server, listener.Addr().String(), testFileName
}
