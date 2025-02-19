# check_locks

**`check_locks.exe`** is a **Windows command-line utility** written in **Go** that detects **locked files and folders**. It helps identify folders that are locked due to open processes, such as **Command Prompt, PowerShell, or Windows Explorer**.

---

## 🚀 Features
✔ **Detects locked files and folders**  
✔ **Exits immediately upon detecting the first lock**  
✔ **Excludes specific subfolders from scanning**  
✔ **Command-line interface (CLI) with `-help` option**  
✔ **Fast execution with minimal system impact**  

---

## 📥 Installation (Build from Source)

### **Prerequisites**
- **Go (Golang) installed** ([Download](https://go.dev/dl/))
- **Git installed** ([Download](https://git-scm.com/downloads))

### **Clone the Repository**
```sh
git clone https://github.com/YOUR_GITHUB_USERNAME/check_locks.git
cd check_locks
