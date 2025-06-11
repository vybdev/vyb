# TODO(vyb): THIS IS YOUR HIGHEST PRIORITY TASK!

## What you will do
Right now, the application only support OpenAI as a model provider. 
We have a stub for Gemini, but no actual provider implementation. 

Both GPT and Reasoning model families should map to the following models:
- Small: "gemini-2.5-flash-preview-05-20"
- Large: "gemini-2.5-pro-preview-06-05"

Use GEMINI_API_KEY env variable for the api key.

Here is an example of Gemini rest call using structured outputs. The `generationConfig` works similarly to the `json_schema` in OpenAI. 
```
curl "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=$GOOGLE_API_KEY" \
-H 'Content-Type: application/json' \
-d '{
      "contents": [{
        "role": "user",
        "parts":[
          { "text": "List a few popular cookie recipes, and include the amounts of ingredients." }
        ]
      }],
      "generationConfig": {
        "responseMimeType": "application/json",
        "responseSchema": {
          "type": "ARRAY",
          "items": {
            "type": "OBJECT",
            "properties": {
              "recipeName": { "type": "STRING" },
              "ingredients": {
                "type": "ARRAY",
                "items": { "type": "STRING" }
              }
            },
            "propertyOrdering": ["recipeName", "ingredients"]
          }
        }
      }
}' 2> /dev/null | head
```

## How you will do it
Perform the next task listed under "What is left to do" in the order they are listed.
You are expected to accomplish no more and no less than one task at a time.
Mark with an [x] the task you have finished.

## What you need to know

*The Q&A section has been removed for brevity – it has already fulfilled its
purpose during the design discussion.*

## What will it look like
*See previous revision – the high-level design was accepted.*

## What is left to do
- [x] First, evaluate the code in this project, and the task description in "What you will do". Then ask as many questions as you need to have full certainty about what is being asked. Ask your questions under "What you need to know" section.
- [x] Once your questions have been answered, propose a design for your solution. Replace the contents under "What will it look like" with the proposed changes to the system. This is not a list of tasks, it is a vision for the final state of the system to satisfy all the requirements.
- [x] Break the implementation into **atomic steps** and list them below. Each
      step must leave the repo in a compilable & tested state.

- [x] **Add Gemini model mapping tests**  
   • `llm/dispatcher_test.go` – verify `mapGeminiModel` returns the correct
   identifiers for every `(family,size)` pair and errors on unknown size.

- [ ] **Create `llm/internal/gemini` package skeleton**  
   • Directory + `gemini.go` with empty public helpers mirroring the OpenAI
   interface (`GetWorkspaceChangeProposals`, `GetModuleContext`,
   `GetModuleExternalContexts`).  
   • Compile-time build passes (methods return `ErrNotImplemented`).

- [ ] **Implement request/response structs & endpoint constants**  
   • Define `message`, `request`, `generationConfig`, and `geminiResponse`
   types.
   • Include helper for marshalling schema into `generationConfig`.
   • No network call yet – unit tests focus on JSON construction.

- [ ] **Wire HTTP call (non-streaming)**  
   • Implement `callGemini` using `net/http`, building the full URL with the
   `GEMINI_API_KEY` query param.  
   • Add basic error handling for non-200 responses.

- [ ] **Hook up `GetWorkspaceChangeProposals`**  
   • Compose system/user messages, invoke `callGemini`, unmarshal into
   `payload.WorkspaceChangeProposal`.
   • Unit test with `httptest.Server` asserting correct payload.

- [ ] **Hook up `GetModuleContext` & `GetModuleExternalContexts`**  
   • Reuse helper for both additional schemas.  
   • Tests similar to step 5.

- [ ] **Enable logging of request/response pairs**  
   • Same convention as OpenAI (`vyb-gemini-*.json`).

- [ ] **Replace dispatcher stubs**  
   • Update `geminiProvider` methods to delegate to
   `llm/internal/gemini` helpers.  
   • Remove temporary error returns.

- [ ] **Environment variable validation**  
   • Return descriptive error when `GEMINI_API_KEY` is missing.  
   • Unit test for this behaviour.

- [ ] **Extend provider list tests**  
    • Assert `llm.SupportedProviders()` now includes "gemini".

- [ ] **Documentation**  
    • Update `llm/README.md` & root `README.md` with Gemini configuration
    instructions.
