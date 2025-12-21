#!/bin/sh
# This is the core installation part script for b4 Universal.

# Get latest release version from GitHub - ONLY returns version string
get_latest_version() {
    api_url="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"

    version=$(fetch_stdout "$api_url" | grep -o '"tag_name": *"[^"]*"' | head -1 | cut -d'"' -f4)

    if [ -z "$version" ]; then
        print_error "Failed to fetch latest version"
        exit 1
    fi

    echo "$version"
}

# Verify checksum
verify_checksum() {
    file="$1"
    checksum_url="$2"
    checksum_file="${file}.sha256"

    print_info "Downloading SHA256 checksum..."

    if ! fetch_file "$checksum_url" "$checksum_file"; then
        rm -f "$checksum_file"
        return 1
    fi

    if [ ! -s "$checksum_file" ]; then
        rm -f "$checksum_file"
        return 1
    fi

    expected_checksum=$(awk '{print $1}' "$checksum_file")

    if [ -z "$expected_checksum" ]; then
        print_warning "Could not parse checksum from file"
        rm -f "$checksum_file"
        return 1
    fi

    if ! command_exists sha256sum; then
        print_warning "sha256sum not found, skipping verification"
        rm -f "$checksum_file"
        return 1
    fi

    actual_checksum=$(sha256sum "$file" | awk '{print $1}')

    rm -f "$checksum_file"

    if [ "$expected_checksum" = "$actual_checksum" ]; then
        print_success "SHA256 checksum verified: $actual_checksum"
        return 0
    else
        print_error "SHA256 checksum mismatch!"
        print_error "Expected: $expected_checksum"
        print_error "Got:      $actual_checksum"
        return 2
    fi
}

# Download file and verify checksums
download_file() {
    url="$1"
    output="$2"
    version="$3"
    arch="$4"

    print_info "Downloading from: $url"

    if ! fetch_file "$url" "$output"; then
        print_error "Download failed"
        return 1
    fi

    # Construct checksum URL
    file_name="${BINARY_NAME}-linux-${arch}.tar.gz"
    sha256_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/${file_name}.sha256"

    # Try to verify SHA256 checksum
    if verify_checksum "$output" "$sha256_url"; then
        return 0
    elif [ $? -eq 2 ]; then
        print_error "Download verification failed!"
        return 0
    else
        print_warning "No checksum file found - unable to verify download integrity"

        if command_exists sha256sum; then
            local_hash=$(sha256sum "$output" | awk '{print $1}')
            print_info "Local SHA256: $local_hash"
        fi
    fi

    return 0
}
