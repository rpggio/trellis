# Generalized Record API Core requirements

- Agent activities like [search, browse, tidy] can be done without loading record-adjacent context. 

- Reasoning on a record requires adjacent context to be loaded.
  - Parent record is required
  - Child context varies on state:
    - LATER | RESOLVED | DISCARDED: load the reference
    - OPEN: load the full content
      - children of open children (grand-children) are loaded as references

- Record workflow state is OPEN | LATER | RESOLVED | DISCARDED

- To reason with a record, the record should be activated. This operation returns the minimum reasoning context. (sync-down)

- A record reference includes the ID, title, and summary

- Record activation is tracked per-session
  - The content for a record is not loaded twice within the same session
    - If the record has changed since last command, they system may return a change event with diff, or just return the entire changed record

- A session may be branched into a new session (typically in a new chat). The branched session retains the active context from original session. The branch operation also performs sync-down.

- When a chat is being ended, the user is expected to close the session (also triggers sync-up). 

- Open sessions are treated as needing resolution, since the chat may contain unsaved information. The user is reminded of open sessions when starting a new session or doing review. The user would need to open the chat in order to close the session.

- Our modeling of this system should consider the location of thoughts relative to sessions. An open session must be assumed to have unsaved info, as the user can converse with the chat agent without our system knowing.

- The MCP system is designed for one chat at a time relating to any given record. 
  - The system should warn the user when activating records that are already active in another session.
  - The system should demand an override from the user before updating records active in another session.

- Concurrent reasoning against records will require a system with more real-time control than MCP: plugins, hooks, or a dedicated chat app. [DEFER]

