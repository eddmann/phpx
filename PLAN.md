# Plan: Fix Tool Working Directory

## Problem Summary

When running `./bin/phpx tool phpstan -- analyze src/`, the tool reports:

```
Path /Users/edd/.phpx/tools/phpstan-phpstan-2.1.33/src does not exist
```

**Root cause:** The `RunTool` function sets the working directory to the tool's installation directory (`~/.phpx/tools/phpstan-phpstan-X.X.X/`), but users expect the tool to operate on files relative to their **current working directory** where they invoked `phpx`.

### Current Behavior (Bug)

```
User CWD: /Users/edd/Projects/my-project
Command:  phpx tool phpstan -- analyze src/

Tool runs with:
  cmd.Dir = /Users/edd/.phpx/tools/phpstan-phpstan-2.1.33/

PHPStan looks for: /Users/edd/.phpx/tools/phpstan-phpstan-2.1.33/src  ← WRONG
User expects:      /Users/edd/Projects/my-project/src                  ← CORRECT
```

### Expected Behavior

The tool should run with the user's current working directory preserved, allowing relative paths like `src/` to resolve correctly.

## Affected Code Location

**`internal/exec/runner.go:30-45`** - `RunTool()` function:

```go
func RunTool(phpPath, toolDir, binary string, args []string) (int, error) {
    binaryPath := filepath.Join(toolDir, "vendor", "bin", binary)

    cmdArgs := []string{binaryPath}
    cmdArgs = append(cmdArgs, args...)

    cmd := exec.Command(phpPath, cmdArgs...)
    cmd.Dir = toolDir  // ← BUG: This changes CWD to tool installation dir
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    err := cmd.Run()
    return exitCode(err), nil
}
```

### Contrast with `RunScript()`

The `RunScript()` function in the same file (`runner.go:10-28`) does NOT set `cmd.Dir`, which means it correctly inherits the caller's working directory. This is the correct behavior.

## Solution

Remove the `cmd.Dir = toolDir` line from `RunTool()`. The tool binary path is already absolute, so there's no need to change the working directory.

### Why Was `cmd.Dir` Set?

Looking at the code, there's no clear reason why `cmd.Dir` was set. Possible theories:

1. **Mistaken assumption** that PHP tools need to run from their installation directory
2. **Copy-paste error** from a different implementation pattern
3. **Deliberate but incorrect** attempt to handle relative path issues

Since the binary path is absolute (`filepath.Join(toolDir, "vendor", "bin", binary)`), there's no need for `cmd.Dir`.

### Edge Cases to Consider

1. **Tools that write files** - Some tools might write output files. With the fix, they'll write to the user's CWD (which is correct behavior, matching how composer global tools work).

2. **Tools that look for config files** - Tools like PHPStan look for `phpstan.neon` in the CWD. This is the expected behavior and will now work correctly.

3. **Tools with hardcoded relative paths** - Unlikely, and would be a tool bug, not a phpx bug.

## Files to Modify

| File | Change |
|------|--------|
| `internal/exec/runner.go` | Remove `cmd.Dir = toolDir` from `RunTool()` |

## The Fix

**`internal/exec/runner.go:30-45`:**

```go
// RunTool executes a tool binary from its installation directory.
func RunTool(phpPath, toolDir, binary string, args []string) (int, error) {
    binaryPath := filepath.Join(toolDir, "vendor", "bin", binary)

    cmdArgs := []string{binaryPath}
    cmdArgs = append(cmdArgs, args...)

    cmd := exec.Command(phpPath, cmdArgs...)
    // Do NOT set cmd.Dir - the tool should run in the user's current directory
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    err := cmd.Run()
    return exitCode(err), nil
}
```

## Verification

After implementation:

```bash
cd /some/project/with/php/code
phpx tool phpstan -- analyze src/
```

Should analyze files in `/some/project/with/php/code/src/` instead of failing with a "path does not exist" error.

## Risk Assessment

**Low risk.** This is a one-line deletion that aligns behavior with:
- How `RunScript()` works in the same file
- How `composer global` tools work
- User expectations for CLI tools

## Testing

While there are no existing tests for `RunTool`, manual verification should confirm:

1. `phpx tool phpstan -- analyze src/` works when run from a project directory
2. Tools can find their config files (e.g., `phpstan.neon`) in the project directory
3. Tools can write output files to the current directory
