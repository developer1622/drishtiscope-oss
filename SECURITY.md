# Security Policy — DrishtiScope

DrishtiScope (दृष्टिScope) is committed to ensuring the highest standards of security, privacy, and system integrity. Because DrishtiScope interacts closely with the Linux kernel and processes, we treat security vulnerabilities with high urgency.

---

## 🛡️ Supported Versions

We provide security updates for the following versions:

| Version | Supported |
| :--- | :--- |
| `1.x` | :white_check_mark: |
| `< 1.0` | :x: |

---

## 🔒 Security Architecture & Guarantees

DrishtiScope is built with privacy and security by design:

1. **Zero Payload Sniffing**: DrishtiScope does **not** intercept, decrypt, or record TLS payloads, model prompts, completions, or LLM token strings. It strictly observes process and kernel metadata (syscalls, file paths, socket states, and CPU/memory vitals).
2. **Loopback by Default**: The Go server binds to `127.0.0.1:8080` by default. Binding to `0.0.0.0` without setting `AUTH_TOKEN` generates explicit warning logs and rejects unauthenticated remote connections with HTTP 403.
3. **Safe Memory Management**: Go memory safety ensures freedom from buffer overflows, use-after-free, and memory corruption in the userspace daemon.
4. **Verified eBPF Programs**: All eBPF programs pass the in-kernel BPF verifier, guaranteeing bounded execution, safe pointer arithmetic, and zero kernel panics.
5. **Zero Hardcoded Secrets**: No API keys, tokens, or credentials are ever embedded in source code. All LLM API keys (`GEMINI_API_KEY`, `OPENAI_API_KEY`, `GROQ_API_KEY`, `ANTHROPIC_API_KEY`) and the optional `AUTH_TOKEN` are loaded exclusively via environment variables. See `.env.example` for a complete template. If a key is absent, the corresponding LLM provider is silently skipped and the built-in rule engine is used as fallback.

---

## 🚨 Reporting a Vulnerability

If you discover a potential security vulnerability in DrishtiScope, please **do not open a public GitHub issue**. Instead, report it privately through GitHub Security Advisories:

1. Navigate to the repository's **Security** tab.
2. Click on **Advisories** -> **Report a vulnerability**.
3. Include:
   - Detailed description of the vulnerability and affected components.
   - Proof-of-concept steps to reproduce.
   - Potential impact on the host system or monitored processes.
4. Our maintainers will acknowledge receipt within 48 hours and coordinate a fix and advisory.
