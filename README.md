# curator-dispatcher

The launch dispatcher for agent systems. It accepts launch requests from people, orchestrators and tools, decides whether this machine can take the run now, provisions a per-agent OS account and its credential binding, starts the agent's executor under that account, tracks the run to its end, and reconciles after crashes.

It works with two privileged and one credential component, each kept narrow:

- [curator-host-helper](https://github.com/relux-works/curator-host-helper) creates and retires per-agent OS accounts; its launcher starts one approved executor under one agent account;
- [curator-credential-broker](https://github.com/relux-works/curator-credential-broker) binds the agent's account and leases it the subscription credential its grant allows.

**Status: design in progress.** Nothing is implemented yet. The specification (v0: a command run under the dispatcher's own service account; v1: a daemon with a socket for orchestrators and the session host, on the same core) is [`spec/dispatcher.md`](spec/dispatcher.md); diagrams are in [`diagrams/`](diagrams/).

## Principles

- The dispatcher runs under its own unprivileged service account; root stays in the helper and the launcher.
- Every launch is one durable record keyed by the caller's request id; repeating a request returns the same run.
- A caller gets only what its own grant allows; the dispatcher narrows it further for the agent.
- Accounting and admission belong here; holding processes moves to the session host when one exists.
- After any crash, the dispatcher reconciles its records with the helper, the broker and the process table before it accepts new work.

## License

Apache License 2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE). Authors: Ivan Oparin and Alexey Grigorev.
