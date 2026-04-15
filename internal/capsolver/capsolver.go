package capsolver

import (
	"github.com/CodeAfu/go-ducc-api/internal/adapters/env/envutils"
	capsolver "github.com/capsolver/capsolver-go"
)

func SolveCaptchaV2Task(siteKey, pageURL string) (string, error) {
	capSolverApiKey, err := envutils.GetString("CAPSOLVER_API_KEY")
	if err != nil {
		return "", err
	}

	capSolver := capsolver.CapSolver{ApiKey: capSolverApiKey}
	solution, err := capSolver.Solve(map[string]any{
		"type":       "ReCaptchaV2TaskProxyless",
		"websiteURL": pageURL,
		"websiteKey": siteKey,
	})
	if err != nil {
		return "", err
	}

	token := solution.Solution.GRecaptchaResponse
	return token, nil
}
