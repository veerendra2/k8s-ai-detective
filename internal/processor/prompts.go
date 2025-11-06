package processor

// TODO Move these separate template files
const (
	promptTpl1 = `
The following JSON, {{ .AlertJSON }}, contains details of an alert from Alertmanager. Use the information in the "labels" field such as "namespace", "pod", "container", and other relevant identifiers to investigate the issue. Perform only basic diagnostic reasoning: review logs, events, and errors related to the affected resources. Do not make any modifications, write any data, or ask for user confirmation. Identify the affected resources, check recent logs and Kubernetes events for errors, warnings, or anomalies, and determine the likely cause of the alert. Output only a short summary of your findings, no more than three to five lines. Do not include any reasoning, explanation of your thought process, or markdown formatting. The output must be concise, factual, and directly actionable, describing only the key issues or anomalies related to the alert.
`

// 	promptTpl2 = `
// ### Context
// The following JSON contains details of an alert from Alertmanager. Use the information in the "labels" field - such as "namespace", "pod", "container", and other relevant identifiers - to investigate the issue. Perform only basic diagnostic reasoning: review logs, events, and errors related to the affected resources. Do NOT make any modifications, write any data, or ask for user confirmation.

// ### Alert JSON
// {{ .AlertJSON }}

// ### Instructions
// 1. Identify the affected resources using the labels provided (e.g., "namespace", "pod", "container").
// 2. Check for:
//    - Recent logs or errors for the affected pod or container.
//    - Kubernetes events in the same namespace.
//    - Any warnings, anomalies, or failures that explain the alert.
// 3. Based on these checks, determine the likely cause of the alert.

// ### Output Requirements
// - Output only a **short summary (3–5 lines maximum)** describing the key findings or probable cause.
// - Do **not** include any reasoning steps, explanations of what you checked, or internal thoughts.
// - Do **not** output sections, lists, or markdown formatting — only plain text.
// - The output must be concise, factual, and directly actionable.

// ### Expected Output
// A short, plain-text summary (just a few lines) describing the most important errors, warnings, or anomalies found in relation to the alert.
// `
)
