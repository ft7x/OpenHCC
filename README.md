# OpenHCC | Hotmail & Country Cap Checker
OpenHCC is a high-performance, concurrent account validator designed for Microsoft/Hotmail services. Built in Go, it leverages a worker-pool architecture to maximize throughput while maintaining a minimal memory footprint.
## 🚀 Key Features
 * High-Speed Validation: Built-in worker pool with reusable HTTP clients to minimize TLS handshake overhead.
 * Real-Time Dashboard: CLI interface providing live CPM (Checks Per Minute), hit rate, and progress tracking.
 * Fading Live Feed: Visual real-time log of successes, 2FA challenges, and locked accounts.
 * Memory Optimized: Efficient response body parsing (capped at 50KB) to prevent RAM bloating on large combo lists.
 * Automated Results: Successes and 2FA accounts are automatically sorted into hits.txt and 2fa.txt.
## 🛠️ Setup & Installation
 * Install Go: Ensure you have Go installed on your system.
 * Clone/Copy the Code: Save the provided script as main.go.
 * Run the Tool:
   go run main.go

 * Configuration:
   * Enter the path to your combo list (format: email:password).
   * Specify the number of concurrent threads (workers).
## 📊 Dashboard Overview
The CLI dashboard provides a structured view of your session:
 * Progress: Percentage of the list completed.
 * CPM: Real-time speed indicator.
 * Live Feed: Displays the last 7 results with color-coded status badges.
 * Hit Counter: Tracks HITS, 2FA, and LOCKED accounts separately.
## ⚠️ Important Legal & GitHub Notice
> [!IMPORTANT]
> Educational Purposes Only
> This tool is provided for educational, research, and authorized security testing purposes only. By using this software, you agree that you are solely responsible for your actions.
> Notice to GitHub / Service Providers:
> This repository is a proof-of-concept demonstrating Go-based network concurrency and CLI UI design. It does not contain or distribute leaked data, credentials, or illegal content. It is intended for security professionals to audit the strength of their own infrastructure and for educational demonstration of the "OpenHCC" protocol.
> 
👤 Credits
 * Developer: FT7
 * Core Engine: OpenHCC Protocol
