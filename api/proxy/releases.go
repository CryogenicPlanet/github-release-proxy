package serverless

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cryogenicplanet/github-release-proxy/shared"

	"github.com/google/go-github/v39/github"
	"golang.org/x/oauth2"
)

func UpdateHandler(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query()
	owner := query.Get("owner")
	repo := query.Get("repo")
	os := query.Get("os")
	arch := query.Get("arch")
	// issue := query.Get("issue")

	if owner == "" || repo == "" || os == "" || arch == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Invalid params")
		return
	}

	token, err := shared.GetInstallationToken(owner)

	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err)
		return
	}

	tokenCtx := context.Background()

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token.GetToken()},
	)
	tc := oauth2.NewClient(tokenCtx, ts)

	tokenClient := github.NewClient(tc)

	repository, resp, err := tokenClient.Repositories.GetLatestRelease(tokenCtx, owner, repo)

	if err != nil || resp.StatusCode >= 400 {
		fmt.Println("Failed to get latest release", resp.StatusCode, err)
		w.WriteHeader(resp.StatusCode)
		fmt.Fprintln(w, err)
		return
	}

	for _, asset := range repository.Assets {
		// fmt.Println("Repo asset", *asset.BrowserDownloadURL)

		if strings.Contains(strings.ToLower(*asset.BrowserDownloadURL), strings.ToLower(os)) && strings.Contains(strings.ToLower(*asset.BrowserDownloadURL), strings.ToLower(arch)) {
			// Correct asset

			fmt.Println(*asset.URL)

			req, err := tokenClient.NewRequest("GET", *asset.URL, nil)
			req.Header.Set("Accept", "application/octet-stream")
			if err != nil {
				fmt.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, err)
				return
			}

			res, err := tokenClient.Client().Do(req)

			if err != nil {
				fmt.Println(err)
				w.WriteHeader(res.StatusCode)
				fmt.Fprintln(w, err)
				return
			}

			// w.Header().Add("Content-Type", "application/octet-stream")
			// w.Header().Add("Content-Disposition", "attachment; filename="+repo+os+arch+".tar.gz")

			for key, values := range res.Header {
				for _, val := range values {
					w.Header().Add(key, val)
				}
			}
			w.WriteHeader(res.StatusCode)

			fmt.Println("Returning data")
			io.Copy(w, res.Body)
			res.Body.Close()

			fmt.Println("Returned")
			return
		}
	}

	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintln(w, "Failed to get release")

}
