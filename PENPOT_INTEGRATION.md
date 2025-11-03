# HelixCode PenPot Integration

This document describes the integration between HelixCode and PenPot for design system management and collaboration.

## Overview

The PenPot integration provides:
- **Design System Management**: Centralized design tokens and components
- **Team Collaboration**: Multi-user design review and feedback
- **Asset Export**: Automated export of design assets for development
- **Version Control**: Design versioning and change tracking
- **API Integration**: Programmatic access to design resources

## Prerequisites

### Required Files
- `penpot.txt` - PenPot API authentication token
- `Design/exports/` - Directory containing design assets

### Required Tools
- `curl` - HTTP client for API calls
- `jq` - JSON processor for API responses

## Setup

### 1. Obtain PenPot Token

1. Log into your PenPot account
2. Navigate to Settings → API Tokens
3. Generate a new token with appropriate permissions
4. Save the token to `penpot.txt` in the project root

### 2. Run Integration Setup

```bash
# Make script executable
chmod +x penpot-integration.sh

# Run complete setup
./penpot-integration.sh setup
```

### 3. Manual Design Import

After setup, import designs manually:

1. Access your PenPot project at the provided URL
2. Create design files and components
3. Import assets from `Design/exports/` directory
4. Set up design system with colors, typography, and components

## Integration Features

### Design System Components

#### Color Palette
- Primary, secondary, and accent colors
- Semantic color tokens
- Dark/light theme variants

#### Typography Scale
- Heading hierarchy (H1-H6)
- Body text variants
- Code and monospace fonts
- Responsive type scales

#### Component Library
- Buttons (primary, secondary, tertiary)
- Form controls (inputs, selects, checkboxes)
- Navigation elements
- Cards and containers
- Modal dialogs

### Asset Management

#### Export Structure
```
Design/exports/
├── png/
│   ├── screens/
│   │   ├── desktop-main-workspace.png
│   │   ├── mobile-ai-chat.png
│   │   └── terminal-dashboard.png
│   └── components/
│       ├── buttons-primary.png
│       └── cards-elevated.png
├── pdf/
│   └── documentation/
│       ├── design-system.pdf
│       └── component-library.pdf
└── export-summary.json
```

#### Automated Exports
- Component screenshots for documentation
- Design system specifications
- Color palette exports
- Typography scales

## API Integration

### Available Endpoints

#### Project Management
- `GET /api/rpc/command/get-profile` - User profile
- `POST /api/rpc/command/create-project` - Create project
- `GET /api/rpc/command/get-projects` - List projects

#### File Management
- `POST /api/rpc/command/create-file` - Create design file
- `GET /api/rpc/command/get-file` - Get file data
- `POST /api/rpc/command/update-file` - Update file

#### Asset Management
- `POST /api/rpc/command/upload-file-media-object` - Upload assets
- `GET /api/rpc/command/get-file-media-objects` - List assets

### Example API Usage

```bash
# Test connection
./penpot-integration.sh test

# Create project
curl -X POST \
  -H "Authorization: Token $(cat penpot.txt)" \
  -H "Content-Type: application/json" \
  -d '{"type": "create-project", "name": "HelixCode", "team-id": "default"}' \
  https://design.penpot.app/api/rpc/command/create-project
```

## Workflow Integration

### Development Workflow

1. **Design Creation**: Designers create components in PenPot
2. **Review & Approval**: Team reviews and approves designs
3. **Asset Export**: Automated export of approved designs
4. **Implementation**: Developers implement using exported assets
5. **Validation**: Design-system compliance validation

### Collaboration Features

#### Team Management
- Role-based access control
- Design review workflows
- Comment and feedback system
- Version history and rollback

#### Design Tokens
- Centralized token management
- Automated code generation
- Cross-platform consistency
- Theme variant support

## Configuration

### Environment Variables

```bash
# PenPot instance URL (optional)
PENPOT_BASE_URL=https://design.penpot.app

# Project settings (set via API)
PENPOT_PROJECT_NAME="HelixCode Design System"
PENPOT_TEAM_ID="default"
```

### Integration Settings

#### Design System Configuration
```json
{
  "colors": {
    "primary": "#0066FF",
    "secondary": "#666666", 
    "accent": "#FF3366"
  },
  "typography": {
    "fontFamily": "Inter, system-ui, sans-serif",
    "scale": [12, 14, 16, 18, 20, 24, 30, 36, 48]
  },
  "spacing": {
    "unit": 8,
    "scale": [0, 4, 8, 16, 24, 32, 48, 64, 96]
  }
}
```

## Usage Examples

### Command Line Interface

```bash
# Check integration status
./penpot-integration.sh status

# Test API connection
./penpot-integration.sh test

# Setup complete integration
./penpot-integration.sh setup
```

### Manual Operations

#### Import Design Assets
1. Open PenPot web interface
2. Navigate to your project
3. Use "Import" feature to add design files
4. Organize assets in libraries

#### Export Development Assets
1. Select components for export
2. Choose export format (SVG, PNG, PDF)
3. Download to `Design/exports/` directory
4. Update export manifest

## Troubleshooting

### Common Issues

#### Authentication Errors
- Verify token in `penpot.txt` is valid
- Check token permissions in PenPot settings
- Ensure token hasn't expired

#### API Connection Issues
- Verify network connectivity
- Check PenPot service status
- Validate API endpoint URLs

#### Asset Import Issues
- Check file formats (SVG, PNG, PDF supported)
- Verify file permissions
- Ensure adequate storage space

### Debug Mode

Enable debug output for troubleshooting:

```bash
# Add debug flag to see detailed output
PENPOT_DEBUG=1 ./penpot-integration.sh setup
```

## Security Considerations

### Token Security
- Store `penpot.txt` securely
- Never commit tokens to version control
- Use environment variables in production
- Rotate tokens regularly

### Access Control
- Limit token permissions to minimum required
- Use team-based access controls
- Audit access regularly
- Monitor for unauthorized usage

## Best Practices

### Design System Management

1. **Centralize Tokens**: Use design tokens for consistency
2. **Version Control**: Track design system changes
3. **Documentation**: Maintain design system documentation
4. **Testing**: Validate design implementation

### Collaboration Workflow

1. **Clear Roles**: Define designer and developer responsibilities
2. **Review Process**: Establish design review workflows
3. **Feedback Loops**: Implement efficient feedback mechanisms
4. **Quality Gates**: Set quality standards for design assets

### Integration Maintenance

1. **Regular Updates**: Keep integration scripts current
2. **API Monitoring**: Monitor for API changes
3. **Error Handling**: Implement robust error handling
4. **Backup Strategy**: Backup design assets and configurations

## Support

### Resources
- [PenPot Documentation](https://help.penpot.app/)
- [PenPot API Reference](https://help.penpot.app/technical-guide/api/)
- [Design System Best Practices](https://designsystemsrepo.com/)

### Getting Help
1. Check integration status: `./penpot-integration.sh status`
2. Test connection: `./penpot-integration.sh test`
3. Review logs for error messages
4. Consult PenPot community forums

## Future Enhancements

### Planned Features
- Automated design token generation
- Real-time design-dev synchronization
- Component code generation
- Design system analytics
- Multi-platform asset optimization

### Integration Roadmap
1. **Phase 1**: Basic API integration and asset management
2. **Phase 2**: Automated design token synchronization
3. **Phase 3**: Real-time collaboration features
4. **Phase 4**: Advanced analytics and optimization

This integration establishes a robust foundation for design-development collaboration, ensuring consistency and efficiency across the HelixCode project.