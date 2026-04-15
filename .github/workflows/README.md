# GitHub Actions Workflows

This repository includes automated workflows for creating releases and building binaries.

## Workflows

### 1. Create Pre-Release (`create-prerelease.yml`)

**Purpose**: Creates a pre-release with Linux AMD64 binary only.

**Trigger**: Manual dispatch from GitHub Actions tab

**Inputs**:
- `version`: Pre-release version (e.g., `v0.0.3-alpha.1`)
- `prerelease`: Mark as pre-release (default: true)
- `release_notes`: Custom release notes (optional)

**What it does**:
1. Builds the `mac_address_parser` binary for Linux AMD64
2. Creates a compressed archive with binary and documentation
3. Generates SHA256 checksums
4. Creates a git tag
5. Creates a GitHub release with assets

### 2. Build Multi-Platform (`build-multiplatform.yml`)

**Purpose**: Creates a release with binaries for multiple platforms.

**Trigger**: Manual dispatch from GitHub Actions tab

**Inputs**:
- `version`: Release version (e.g., `v0.0.3-alpha.1`)

**Platforms supported**:
- Linux AMD64/ARM64
- macOS AMD64/ARM64 (Intel/Apple Silicon)
- Windows AMD64

**What it does**:
1. Builds binaries for all supported platforms
2. Creates platform-specific archives (tar.gz for Unix, zip for Windows)
3. Generates SHA256 checksums for all assets
4. Creates a git tag
5. Creates a GitHub release with all platform binaries

## How to Use

### Option 1: Single Platform Pre-Release

1. Go to the **Actions** tab in your GitHub repository
2. Select **"Create Pre-Release"** workflow
3. Click **"Run workflow"**
4. Fill in the inputs:
   - Version: `v0.0.3-alpha.1`
   - Pre-release: ✅ (checked)
   - Release notes: (optional custom notes)
5. Click **"Run workflow"**

### Option 2: Multi-Platform Release

1. Go to the **Actions** tab in your GitHub repository
2. Select **"Build Multi-Platform Binaries"** workflow
3. Click **"Run workflow"**
4. Fill in the version: `v0.0.3-alpha.1`
5. Click **"Run workflow"**

## Generated Assets

Both workflows will create:

- **Binary archives**: Platform-specific compressed files containing:
  - `mac_address_parser` executable
  - `README.md` documentation
  - `mac-address-table-sample.json` sample
- **Checksum files**: `.sha256` files for verification
- **Release notes**: Automated or custom release notes
- **Git tags**: Properly tagged releases

## Requirements

- Repository must have **write permissions** for the GitHub token
- Go 1.21+ (automatically installed by the workflow)
- The `src/SwitchOutput/Cisco/Nexus/10/mac_address_parser/` directory structure
- Uses latest GitHub Actions (checkout@v4, setup-go@v4, upload/download-artifact@v4)

## Notes

- All releases are marked as **pre-releases** by default
- Binaries are built with optimizations (`-ldflags="-w -s"`) for smaller size
- Cross-compilation is enabled for all supported platforms
- Checksums are automatically generated for integrity verification
- Uses the latest GitHub Actions to avoid deprecation warnings
