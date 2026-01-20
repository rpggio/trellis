Complex MCP servers use a layered approach to guide agents, moving beyond simple API descriptions to ensure the model understands not just *how* to call a tool, but *why* and *when*.

### 1. The "User Manual": Server Instructions

A relatively recent advancement in the protocol is the **Server Instructions** capability. Instead of burying logic in individual tool descriptions, a server can provide a global natural language document.

* **Purpose:** It acts as a "ReadMe" for the entire server.
* **Content:** It captures cross-tool relationships (e.g., "Always call `authenticate` before `fetch_data`"), operational constraints (e.g., "Batch requests in groups of 10"), and shared vocabulary.
* **Benefit:** This prevents the agent from making "shallow" calls by providing a unified mental model of the system's state and boundaries.

### 2. The "Interface": Description Fields

The API's `description` fields remain the primary "just-in-time" guidance. However, complex servers use them strategically:

* **Semantic Clues:** Descriptions often include examples of valid input values and mention "side effects" that might not be obvious from the schema (e.g., "This tool invalidates the cache for the 'user_profile' resource").
* **Strict Typing:** They utilize JSON Schema's `oneOf`, `anyOf`, and `enum` constraints to reduce the "fuzzy" reasoning the agent has to do, forcing it into a deterministic path.

---

### 3. Document Resources (Deep Context)

For systems like your **Ganot/Ganos** project, servers often expose **Resources** as foundational documentation.

* **Foundational Framing:** Complex servers may serve a `concepts.md` or `SKILL.md` resource. These are often used to separate "Framing" (the neutral structure of the problem space) from "Claims" (the dynamic logic being applied).
* **Reference Material:** If an agent is performing complex reasoning, the server might point it toward a specific resource URI that contains the "Ground Truth" or a "Reasoning Guide" before the agent attempts to use an execution tool.

---

### 4. Progressive Discovery

Progressive discovery is becoming the standard for complex systems to avoid "context window bloat."

* **Discovery Degree:** The agent typically starts with a `tools/list` request, which returns only names and high-level summaries.
* **Just-in-Time Loading:** The agent only requests the full JSON schema or the detailed "instruction resource" once it identifies a tool is likely relevant to the user's intent.
* **Dynamic Skills:** Some advanced servers (like the "Agent Skills" pattern) expose a single `explore_skills` tool. The agent discovers a library of capabilities and must explicitly "acquire" the detailed instructions for a specific skill, keeping the primary context window lean.

| Mechanism | Scope | Primary Purpose |
| --- | --- | --- |
| **Server Instructions** | Global | Defines the "Rules of Engagement" and system state. |
| **Tool Descriptions** | Tool-specific | Explains immediate intent and parameter usage. |
| **Resource Templates** | Domain-specific | Provides deep documentation or "Framing" data. |
| **Progressive Disclosure** | Interaction-level | Minimizes token usage by loading details on-demand. |
