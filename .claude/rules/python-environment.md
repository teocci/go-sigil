# Python Environment Rules

## Required Python Version

- **Windows:** Python 3.11.9
- **Linux:** Python 3.11.9 (or `python3.11` from package manager)
- **macOS:** Python 3.11.9 (or `python3.11` via Homebrew)

## Virtual Environment Setup

Before running any Python command, ensure the virtual environment exists and is activated.

### Creating the environment (if `.venv/` does not exist)

```bash
# Windows
py -3.11 -m venv .venv

# Linux / macOS
python3.11 -m venv .venv
```

### Activating the environment

Always activate before running Python:

```bash
# Windows (Git Bash / MSYS2)
source .venv/Scripts/activate

# Linux / macOS
source .venv/bin/activate
```

### Installing dependencies

After activation, install required packages:

```bash
pip install --upgrade pip
pip install matplotlib numpy tiktoken
```

## Rules for All Python Execution

1. **Never use the system Python.** Always run Python inside the `.venv` environment.
2. **Always activate first.** Prefix every Bash call that runs Python with the activation command:
   ```bash
   source .venv/Scripts/activate && python benchmark/charts.py benchmark/results
   ```
   On Linux/macOS:
   ```bash
   source .venv/bin/activate && python benchmark/charts.py benchmark/results
   ```
3. **If `.venv/` does not exist**, create it before proceeding. Do not skip this step.
4. **If a package is missing**, install it inside the active environment with `pip install`.
5. **Do not add `.venv/` to version control.** It should be in `.gitignore`.
