# Security Policy — DrishtiScope

DrishtiScope (दृष्टिScope) is committed to ensuring high standards of security, privacy, and system integrity. Because DrishtiScope interacts with operating system kernel interfaces and host processes, security and stability are fundamental priorities.

---

## 🛡️ Supported Versions

Security updates are provided for the following versions:

| Version | Supported |
| :--- | :--- |
| `1.x` | :white_check_mark: |
| `< 1.0` | :x: |

---

## 🔒 Security Architecture & Guarantees

DrishtiScope is designed with privacy, safety, and isolation in mind:

1. **Zero Payload Inspection**: DrishtiScope does **not** intercept, decrypt, or record network payloads, user documents, prompt text, or confidential application contents. It strictly observes system metadata (system calls, file paths, socket endpoints/states, and CPU/memory vitals).
2. **Loopback by Default**: The daemon binds strictly to `127.0.0.1:8080` by default. Binding to public interfaces (`0.0.0.0`) without setting `AUTH_TOKEN` generates explicit warning logs and rejects unauthenticated remote connections.
3. **Memory Safety**: The userspace daemon is written in Go, providing automatic bounds checking and safe memory handling.
4. **Verified eBPF Programs**: All in-kernel eBPF programs are verified by the Linux kernel BPF verifier prior to loading, guaranteeing bounded loops, safe memory access, and zero kernel instability.
5. **Zero Hardcoded Secrets**: No API keys, passwords, or credentials are ever embedded in the repository. All optional diagnostic credentials and authentication tokens are loaded strictly via environment variables at runtime.

---

## 🚨 Reporting a Vulnerability

If you discover a potential security vulnerability in DrishtiScope, please **do not open a public GitHub issue**. Instead, report it privately through GitHub Security Advisories:

1. Navigate to the repository's **Security** tab.
2. Click on **Advisories** -> **Report a vulnerability**.
3. Include:
   - Detailed description of the vulnerability and affected components.
   - Proof-of-concept steps to reproduce.
   - Potential impact on the host system or monitored processes.
4. Maintainers will review the submission promptly and coordinate a fix.
