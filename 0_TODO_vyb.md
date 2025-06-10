# TODO(vyb): THIS IS YOUR HIGHEST PRIORITY TASK!

## What you will do
Right now the application has an llm package, but inside it we only have an openai implementation that is exported and called directly by the business logic that requires the functionality. Additionally, even the templated commands have a field for model name, which is directly tied to a model provider.

A more robust and flexible solution would be for the model providers to be completely abstracted away from the business logic. All the exported code should be at the llm package, and based on user configuration, the code then decides whether to delegate the calls to OpenAi, or any of the other--to be supported--model providers.

Instead of calling out the model name directly in the commands and templates, we could refer to them as model families (GPT and reasoning), and model size (Large,Small). And let the provider-specific code map these concepts to its own domain.

The configuration should come from .vyb/config.yaml. This file should be created during vyb init execution, and default the LLM-provider to OpenAI.

## How you will do it
Perform the next task listed under "What is left to do" in the order they are listed. 
You are expected to accomplish no more and no less than one task at a time.
Mark with an [x] the task you have finished.

## What you need to know

- Question: Your question should be formatted like this. 
  - Answer: And my answer will be formatted like this.

## What will it look like
This section will contain your proposed solution for the problem that you were given. 

## What is left to do
- [ ] First, evaluate the code in this project, and the task description in "What you will do". Then ask as many questions as you need to have full certainty about what is being asked. Ask your questions under "What you need to know" section.
- [ ] Once your questions have been answered, propose a design for your solution. Replace the contents under "What will it look like" with the proposed changes to the system. This is not a list of tasks, it is a vision for the final state of the system to satisfy all the requirements.
- [ ] Now review everything you know about this task, and break it down into a list of atomic changes, and add them to this list here. Each change should be selfcontained, and leave the system one step closer to the desired state. Make sure to include tests and documentation changes alongside each step, since the repository should not get into an inconsistent state in between these changes. 
