# Get geosite path from config using jq if available
get_geosite_path_from_config() {
    if [ -f "$CONFIG_FILE" ] && command_exists jq; then
        geosite_path=$(jq -r '.system.geo.geosite_path // empty' "$CONFIG_FILE" 2>/dev/null)
        if [ -n "$geosite_path" ] && [ "$geosite_path" != "null" ]; then
            # Extract directory from path
            echo "$(dirname "$geosite_path")"
            return 0
        fi
    fi
    return 1
}

# Display geosite source menu and get user choice
select_geo_source() {
    echo "" >&2
    echo "=======================================" >&2
    echo "  Select Geosite Data Source" >&2
    echo "=======================================" >&2
    echo "" >&2

    # Display sources (using POSIX-compliant iteration)
    echo "$GEODAT_SOURCES" | while IFS='|' read -r num name url; do
        if [ -n "$num" ]; then
            printf "  ${GREEN}%s${NC}) %s\n" "$num" "$name" >&2
        fi
    done

    echo "" >&2
    printf "${CYAN}Select source (1-5) or 'q' to skip: ${NC}" >&2
    read choice

    case "$choice" in
    [qQ] | [qQ][uU][iI][tT])
        return 1
        ;;
    [1-5])
        # Extract URL for selected choice (POSIX-compliant)
        selected_url=$(echo "$GEODAT_SOURCES" | grep "^${choice}|" | cut -d'|' -f3)
        if [ -n "$selected_url" ]; then
            echo "$selected_url"
            return 0
        else
            print_error "Invalid selection"
            return 1
        fi
        ;;
    *)
        print_error "Invalid selection"
        return 1
        ;;
    esac
}

download_geodat() {
    geosite_url="$1/geosite.dat"
    geoip_url="$1/geoip.dat"

    save_dir="$2"

    # Verify save_dir is writable, fallback if needed
    if [ ! -w "$(dirname "$save_dir")" ] && [ ! -d "$save_dir" ]; then
        if [ -d "/opt/etc" ] && [ -w "/opt/etc" ]; then
            save_dir="/opt/etc/b4"
            print_warning "Original path not writable, using: $save_dir"
        fi
    fi

    geosite_file="${save_dir}/geosite.dat"
    geoip_file="${save_dir}/geoip.dat"

    print_info "Downloading geosite.dat and geoip.dat from: $geosite_url"

    # Create directory if it doesn't exist
    if [ ! -d "$save_dir" ]; then
        mkdir -p "$save_dir" || {
            print_error "Failed to create directory: $save_dir"
            return 1
        }
    fi

    # Download the file
    if command_exists wget; then
        wget_opts="-q"
        wget $wget_opts -O "$geosite_file" "$geosite_url" || {
            print_error "Download failed"
            return 1
        }
    elif command_exists curl; then
        curl -L -# -o "$geosite_file" "$geosite_url" || {
            print_error "Download failed"
            return 1
        }
    else
        print_error "Neither wget nor curl found"
        return 1
    fi

    # Download the file
    if command_exists wget; then
        wget_opts="-q"
        wget $wget_opts -O "$geoip_file" "$geoip_url" || {
            print_error "Download failed"
            return 1
        }
    elif command_exists curl; then
        curl -L -# -o "$geoip_file" "$geoip_url" || {
            print_error "Download failed"
            return 1
        }
    else
        print_error "Neither wget nor curl found"
        return 1
    fi

    # Verify file was downloaded and is not empty
    if [ ! -f "$geosite_file" ] || [ ! -s "$geosite_file" ]; then
        print_error "Downloaded file is empty or missing"
        rm -f "$geosite_file"
        return 1
    fi

    # Verify file was downloaded and is not empty
    if [ ! -f "$geoip_file" ] || [ ! -s "$geoip_file" ]; then
        print_error "Downloaded file is empty or missing"
        rm -f "$geoip_file"
        return 1
    fi

    print_success "Geosite file downloaded to: $geosite_file"
    print_success "GeoIP file downloaded to: $geoip_file"
    return 0
}

# Update config file with geodat paths
update_config_geodat_path() {
    geosite_file="$1"
    geoip_file="$2"
    local site_url="$3/geosite.dat"
    local ip_url="$3/geoip.dat"

    # Try to update with jq if available
    if command_exists jq; then
        print_info "Updating config file..."

        if [ ! -f "$CONFIG_FILE" ]; then
            jq -n \
                --arg geosite_path "$geosite_file" \
                --arg geosite_url "$site_url" \
                --arg geoip_path "$geoip_file" \
                --arg geoip_url "$ip_url" \
                '{
                    system: {
                        geo: {
                            sitedat_path: $geosite_path,
                            sitedat_url: $geosite_url,
                            ipdat_path: $geoip_path,
                            ipdat_url: $geoip_url
                        }
                    }
                }' >"$CONFIG_FILE"
            print_success "Created new config file with geodat settings"
            return 0
        fi

        # Create temporary file
        temp_file="${CONFIG_FILE}.tmp"

        # Merge into existing geo object instead of replacing
        if jq \
            --arg geosite_path "$geosite_file" \
            --arg geosite_url "$site_url" \
            --arg geoip_path "$geoip_file" \
            --arg geoip_url "$ip_url" \
            '.system.geo = (.system.geo // {}) + {
                 sitedat_path: $geosite_path,
                 sitedat_url: $geosite_url,
                 ipdat_path: $geoip_path,
                 ipdat_url: $geoip_url
             }' \
            "$CONFIG_FILE" >"$temp_file" 2>/dev/null; then

            mv "$temp_file" "$CONFIG_FILE" || {
                print_error "Failed to update config file"
                rm -f "$temp_file"
                return 1
            }
            print_success "Config updated:"
            print_success "  Geosite: $geosite_file"
            print_success "  URL: $site_url"
            print_success "  GeoIP:   $geoip_file"
            print_success "  URL: $ip_url"

            # Show what was actually written
            print_info "Verifying config..."
            if command_exists jq; then
                jq '.system.geo' "$CONFIG_FILE" 2>/dev/null || true
            fi
            return 0
        else
            print_error "Failed to parse config with jq"
            rm -f "$temp_file"
            return 1
        fi
    else
        print_warning "jq not found - cannot automatically update config"
        print_info "Please manually add to your config file:"
        print_info '  "system": {'
        print_info '    "geo": {'
        print_info "      \"sitedat_path\": \"$geosite_file\","
        print_info "      \"sitedat_url\": \"$site_url\","
        print_info "      \"ipdat_path\": \"$geoip_file\","
        print_info "      \"ipdat_url\": \"$ip_url\""
        print_info '    }'
        print_info '  }'
        echo ""
        print_info "Or update paths in B4 Web UI: Settings -> Geodat Settings"
        return 0
    fi
}

# Setup geosite data
setup_geodat() {
    echo ""
    echo "======================================="
    echo "  GEO Data Setup"
    echo "======================================="
    echo ""

    if [ -z "$GEOSITE_SRC" ] && [ -z "$GEOSITE_DST" ]; then
        # Ask if user wants to download geosite
        printf "${CYAN}Do you want to download geosite.dat & geoip.dat files? (y/N): ${NC}"
        read answer
    else
        answer="y"
    fi

    case "$answer" in
    [yY] | [yY][eE][sS])
        # Select source
        if [ -z "$GEOSITE_SRC" ]; then
            geosite_url=$(select_geo_source)
            if [ $? -ne 0 ] || [ -z "$geosite_url" ]; then
                print_info "Geosite setup skipped"
                return 0
            fi
        else
            geosite_url="$GEOSITE_SRC"
            print_info "Using geosite source: $geosite_url"
        fi

        if [ -z "$GEOSITE_DST" ]; then

            # Get save directory
            default_dir="$CONFIG_DIR"

            # Try to get existing path from config
            existing_dir=$(get_geosite_path_from_config || true)
            if [ -n "$existing_dir" ]; then
                default_dir="$existing_dir"
                print_info "Found existing geosite path in config: $default_dir"
            fi

            echo ""
            printf "${CYAN}Save directory [${default_dir}]: ${NC}"
            read geosite_dst_dir

            # Use default if empty
            if [ -z "$geosite_dst_dir" ]; then
                geosite_dst_dir="$default_dir"
            fi
        else
            geosite_dst_dir="$GEOSITE_DST"
            print_info "Using geodat destination: $geosite_dst_dir"
        fi

        # Download geosite file
        download_geodat "$geosite_url" "$geosite_dst_dir"
        geosite_file="${geosite_dst_dir}/geosite.dat"
        geoip_file="${geosite_dst_dir}/geoip.dat"

        # Update config
        update_config_geodat_path "$geosite_file" "$geoip_file" "$geosite_url"

        print_success "Geosite setup completed!"
        return 0

        ;;
    *)
        print_info "Geosite setup skipped"
        ;;
    esac

    echo ""
    return 0
}
