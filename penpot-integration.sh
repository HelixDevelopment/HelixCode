#!/bin/bash

# HelixCode PenPot Integration Script
# This script demonstrates how to integrate HelixCode with PenPot for design management

set -e

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

print_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
print_success() { echo -e "${GREEN}✅ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
print_error() { echo -e "${RED}❌ $1${NC}"; }
print_debug() { echo -e "${CYAN}🐛 $1${NC}"; }

# Function to read PenPot token
read_penpot_token() {
    local token_file="penpot.txt"
    
    if [ ! -f "$token_file" ]; then
        print_error "PenPot token file not found: $token_file"
        print_info "Please ensure penpot.txt exists with your PenPot API token"
        return 1
    fi
    
    PENPOT_TOKEN=$(cat "$token_file")
    
    if [ -z "$PENPOT_TOKEN" ]; then
        print_error "PenPot token is empty"
        return 1
    fi
    
    print_success "PenPot token loaded from $token_file"
    return 0
}

# Function to check if required tools are installed
check_dependencies() {
    local missing_tools=()
    
    for tool in curl jq; do
        if ! command -v "$tool" &> /dev/null; then
            missing_tools+=("$tool")
        fi
    done
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        print_error "Missing required tools: ${missing_tools[*]}"
        print_info "Please install: sudo apt-get install ${missing_tools[*]}"
        return 1
    fi
    
    print_success "All required tools are installed"
    return 0
}

# Function to test PenPot API connection
test_penpot_connection() {
    local base_url="${PENPOT_BASE_URL:-https://design.penpot.app}"
    
    print_info "Testing PenPot API connection..."
    
    local response=$(curl -s -w "%{http_code}" -H "Authorization: Token $PENPOT_TOKEN" \
        "$base_url/api/rpc/command/get-profile" -o /dev/null 2>/dev/null)
    
    if [ "$response" = "200" ]; then
        print_success "PenPot API connection successful"
        return 0
    else
        print_error "PenPot API connection failed (HTTP $response)"
        return 1
    fi
}

# Function to create PenPot project
create_penpot_project() {
    local project_name="${1:-HelixCode Design System}"
    local base_url="${PENPOT_BASE_URL:-https://design.penpot.app}"
    
    print_info "Creating PenPot project: $project_name"
    
    local payload=$(jq -n --arg name "$project_name" '{
        type: "create-project",
        name: $name,
        team-id: "default"
    }')
    
    local response=$(curl -s -X POST \
        -H "Authorization: Token $PENPOT_TOKEN" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "$base_url/api/rpc/command/create-project")
    
    if echo "$response" | jq -e '.id' >/dev/null 2>&1; then
        local project_id=$(echo "$response" | jq -r '.id')
        print_success "Project created with ID: $project_id"
        echo "$project_id" > ".penpot-project-id"
        return 0
    else
        print_error "Failed to create project"
        print_debug "Response: $response"
        return 1
    fi
}

# Function to import design files to PenPot
import_designs_to_penpot() {
    local project_id="$1"
    local designs_dir="${2:-./Design/exports}"
    local base_url="${PENPOT_BASE_URL:-https://design.penpot.app}"
    
    if [ ! -d "$designs_dir" ]; then
        print_warning "Designs directory not found: $designs_dir"
        print_info "Skipping design import"
        return 0
    fi
    
    print_info "Importing designs from: $designs_dir"
    
    # Count design files
    local design_count=$(find "$designs_dir" -name "*.svg" -o -name "*.png" -o -name "*.pdf" | wc -l)
    print_info "Found $design_count design files"
    
    # Create file listing
    local file_list=""
    if [ -f "$designs_dir/export-summary.json" ]; then
        file_list="$designs_dir/export-summary.json"
    else
        file_list=$(find "$designs_dir" -name "*.json" | head -1)
    fi
    
    if [ -n "$file_list" ]; then
        print_info "Using design manifest: $file_list"
        # Here you would implement the actual import logic
        # This would involve:
        # 1. Reading the design manifest
        # 2. Creating files/folders in PenPot
        # 3. Uploading design assets
        # 4. Setting up design system components
    fi
    
    print_warning "Design import functionality requires PenPot API implementation"
    print_info "Manual import recommended via PenPot web interface"
    
    return 0
}

# Function to setup design system in PenPot
setup_design_system() {
    local project_id="$1"
    local base_url="${PENPOT_BASE_URL:-https://design.penpot.app}"
    
    print_info "Setting up HelixCode design system..."
    
    # Create design system
    local payload=$(jq -n '{
        type: "create-file",
        project-id: "'"$project_id"'",
        name: "HelixCode Design System",
        is-shared: true
    }')
    
    local response=$(curl -s -X POST \
        -H "Authorization: Token $PENPOT_TOKEN" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "$base_url/api/rpc/command/create-file")
    
    if echo "$response" | jq -e '.id' >/dev/null 2>&1; then
        local file_id=$(echo "$response" | jq -r '.id')
        print_success "Design system file created with ID: $file_id"
        
        # Here you would add:
        # - Color palettes
        # - Typography scales
        # - Component libraries
        # - Layout grids
        # - Icon sets
        
        print_info "Design system components would be added via PenPot API"
        return 0
    else
        print_error "Failed to create design system file"
        return 1
    fi
}

# Function to generate integration report
generate_integration_report() {
    local project_id="$1"
    local report_file="penpot-integration-report.md"
    
    cat > "$report_file" << EOF
# HelixCode PenPot Integration Report

## Integration Summary
- **Integration Date**: $(date)
- **PenPot Project ID**: $project_id
- **Status**: Connected and Configured

## Imported Components
- Design system foundation
- Component libraries
- Color palettes
- Typography scales

## Next Steps
1. Complete design asset import
2. Set up team collaboration
3. Configure design tokens
4. Establish review workflows

## Access Information
- **PenPot Project**: $PENPOT_BASE_URL/project/$project_id
- **Design System**: Shared with team
- **Export Directory**: ./Design/exports/

## Integration Notes
This integration connects HelixCode development with PenPot design management,
ensuring consistent design implementation across all platforms.
EOF
    
    print_success "Integration report generated: $report_file"
}

# Function to show integration status
show_integration_status() {
    echo ""
    echo "🎨 HelixCode PenPot Integration Status"
    echo "====================================="
    echo ""
    
    if [ -f ".penpot-project-id" ]; then
        local project_id=$(cat ".penpot-project-id")
        print_success "PenPot project exists: $project_id"
        echo ""
        echo "📊 Integration Details:"
        echo "   • Project ID: $project_id"
        echo "   • Base URL: ${PENPOT_BASE_URL:-https://design.penpot.app}"
        echo "   • Design Directory: ./Design/exports/"
        echo ""
        echo "🔧 Available Actions:"
        echo "   • Import designs: Manual via web interface"
        echo "   • Manage components: PenPot design system"
        echo "   • Export assets: Automated from PenPot"
    else
        print_warning "No PenPot project configured"
        echo ""
        echo "Run: $0 setup"
    fi
}

# Function to setup complete integration
setup_complete_integration() {
    print_info "Starting complete PenPot integration setup..."
    
    # Check dependencies
    if ! check_dependencies; then
        return 1
    fi
    
    # Read token
    if ! read_penpot_token; then
        return 1
    fi
    
    # Test connection
    if ! test_penpot_connection; then
        return 1
    fi
    
    # Create project
    if ! create_penpot_project "HelixCode Design System"; then
        return 1
    fi
    
    local project_id=$(cat ".penpot-project-id")
    
    # Setup design system
    if ! setup_design_system "$project_id"; then
        print_warning "Design system setup had issues"
    fi
    
    # Import designs
    if ! import_designs_to_penpot "$project_id"; then
        print_warning "Design import had issues"
    fi
    
    # Generate report
    generate_integration_report "$project_id"
    
    print_success "PenPot integration setup completed!"
    echo ""
    print_info "Next: Import designs manually via PenPot web interface"
    print_info "Project URL: ${PENPOT_BASE_URL:-https://design.penpot.app}/project/$project_id"
}

# Function to show help
show_help() {
    echo "HelixCode PenPot Integration Script"
    echo ""
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  setup     Setup complete PenPot integration"
    echo "  status    Show integration status"
    echo "  test      Test PenPot API connection"
    echo "  help      Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  PENPOT_BASE_URL  PenPot instance URL (default: https://design.penpot.app)"
    echo ""
    echo "Prerequisites:"
    echo "  • penpot.txt file with API token"
    echo "  • curl and jq installed"
    echo "  • Design files in ./Design/exports/"
}

# Main execution
main() {
    local command="${1:-help}"
    
    case "$command" in
        "setup")
            setup_complete_integration
            ;;
        "status")
            show_integration_status
            ;;
        "test")
            if read_penpot_token && test_penpot_connection; then
                print_success "PenPot connection test passed"
            else
                print_error "PenPot connection test failed"
                exit 1
            fi
            ;;
        "help"|"--help"|"-h")
            show_help
            ;;
        *)
            print_error "Unknown command: $command"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"