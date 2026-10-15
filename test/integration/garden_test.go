//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:revive // Ginkgo's DSL reads as plain English only without the package name
	. "github.com/onsi/gomega"    //nolint:revive // same for Gomega's matchers
)

type idea struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	PlantedBy string    `json:"plantedBy"`
	PlantedAt time.Time `json:"plantedAt"`
}

func createProject(ctx context.Context, token string) project {
	response := call(ctx, http.MethodPost, "/api/projects", token, map[string]string{"name": "Garden " + time.Now().Format(time.RFC3339Nano)})
	Expect(response).To(HaveHTTPStatus(http.StatusCreated))

	var created project
	Expect(json.NewDecoder(response.Body).Decode(&created)).To(Succeed())

	return created
}

func plant(ctx context.Context, token string, projectID string, text string) idea {
	response := call(ctx, http.MethodPost, "/api/projects/"+projectID+"/ideas", token, map[string]string{"text": text})
	Expect(response).To(HaveHTTPStatus(http.StatusCreated))

	var planted idea
	Expect(json.NewDecoder(response.Body).Decode(&planted)).To(Succeed())

	return planted
}

func gardenOf(ctx context.Context, token string, projectID string) []idea {
	response := call(ctx, http.MethodGet, "/api/projects/"+projectID+"/ideas", token, nil)
	Expect(response).To(HaveHTTPStatus(http.StatusOK))

	var ideas []idea
	Expect(json.NewDecoder(response.Body).Decode(&ideas)).To(Succeed())

	return ideas
}

var _ = Describe("planting an idea", func() {
	Context("in a project you're a member of", func() {
		It("shows the idea in the garden, newest first, with who planted it and when", func(ctx SpecContext) {
			garden := createProject(ctx, accessToken)

			first := plant(ctx, accessToken, garden.ID, "search should find ideas by any word")
			second := plant(ctx, accessToken, garden.ID, "  a calmer color for done tickets  ")

			Expect(second.ID).NotTo(BeEmpty())
			Expect(second.Text).To(Equal("a calmer color for done tickets"))
			Expect(second.PlantedBy).To(Equal("frogly-tests"))
			Expect(second.PlantedAt).To(BeTemporally("~", time.Now(), time.Minute))

			Expect(gardenOf(ctx, accessToken, garden.ID)).To(Equal([]idea{second, first}))
		}, SpecTimeout(10*time.Second))
	})

	Context("in a project you aren't a member of", func() {
		It("neither shows the garden nor takes the idea", func(ctx SpecContext) {
			garden := createProject(ctx, accessToken)
			plant(ctx, accessToken, garden.ID, "only for members")

			listed := call(ctx, http.MethodGet, "/api/projects/"+garden.ID+"/ideas", otherAccessToken, nil)
			Expect(listed).To(HaveHTTPStatus(http.StatusNotFound))

			planted := call(ctx, http.MethodPost, "/api/projects/"+garden.ID+"/ideas", otherAccessToken, map[string]string{"text": "let me in"})
			Expect(planted).To(HaveHTTPStatus(http.StatusNotFound))

			Expect(gardenOf(ctx, accessToken, garden.ID)).To(HaveLen(1))
		}, SpecTimeout(10*time.Second))
	})
})
