#!/bin/bash
set -e

# Generate sample projects and data using the trellis API

usage() {
    echo "Usage: $0 <deployment_directory>"
    echo ""
    echo "Generates sample projects and data in the deployed trellis server."
    echo "The server must be running before executing this script."
    echo ""
    echo "Example:"
    echo "  $0 ~/my-trellis-deployment"
    exit 1
}

if [ $# -ne 1 ]; then
    usage
fi

DEPLOY_DIR="$1"

if [ ! -f "$DEPLOY_DIR/.env" ]; then
    echo "❌ Error: Deployment not found at $DEPLOY_DIR"
    echo "   Run deploy-standalone.sh first"
    exit 1
fi

# Load API key
source "$DEPLOY_DIR/.env"
API_KEY="$TRELLIS_API_KEY"
BASE_URL="http://${TRELLIS_SERVER_HOST:-127.0.0.1}:${TRELLIS_SERVER_PORT:-8080}/rpc"
AUTH_ENABLED="${TRELLIS_AUTH_ENABLED:-true}"

use_auth=true
case "$AUTH_ENABLED" in
    false|FALSE|0|no|NO)
        use_auth=false
        ;;
esac

echo "🎨 Generating sample data for trellis"
echo ""

# Helper function to make API calls
api_call() {
    local method="$1"
    local params="$2"

    if [ "$use_auth" = true ]; then
        curl -s -X POST "$BASE_URL" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $API_KEY" \
            -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"$method\",\"params\":$params}"
    else
        curl -s -X POST "$BASE_URL" \
            -H "Content-Type: application/json" \
            -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"$method\",\"params\":$params}"
    fi
}

# Check if server is running
echo "🔍 Checking server status..."
response=$(api_call "list_projects" "{}")
if echo "$response" | grep -q "error"; then
    echo "❌ Server error or not running:"
    echo "$response"
    echo ""
    echo "Start the server with: cd $DEPLOY_DIR && ./start.sh"
    exit 1
fi
echo "✅ Server is running"
echo ""

# Create a sample project
echo "📁 Creating sample project..."
project_response=$(api_call "create_project" '{
    "id": "mobile-app-redesign",
    "name": "Mobile App Redesign",
    "description": "Design exploration for the next generation mobile application focusing on user experience and performance"
}')
echo "   Created: Mobile App Redesign"
echo ""

# Create root-level design question
echo "📝 Creating sample records..."
record1=$(api_call "create_record" '{
    "parent_id": null,
    "type": "question",
    "title": "How should we approach the navigation redesign?",
    "summary": "Exploring whether to use bottom navigation, drawer, or hybrid approach for main app navigation",
    "body": "The current navigation structure uses a side drawer, but user research indicates confusion about accessing key features. We need to determine the optimal navigation pattern that balances discoverability with screen real estate.\n\nKey considerations:\n- 60% of users are one-handed mobile users\n- Core features: Dashboard, Messages, Profile, Settings, Search\n- Need to support both iOS and Android conventions\n- Accessibility requirements for navigation announcements\n\nInitial analysis suggests bottom navigation might improve task completion times, but we need to validate this against our specific use cases and user segments."
}' | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "   ✓ Created root question: Navigation redesign"

# Create conclusion for the navigation question
record2=$(api_call "create_record" '{
    "parent_id": "'$record1'",
    "type": "conclusion",
    "title": "Use adaptive bottom navigation with collapsible sections",
    "summary": "Bottom nav for primary actions (4 items), with contextual overflow menu for secondary features",
    "body": "After analyzing user flows and conducting A/B testing with prototypes, we recommend an adaptive bottom navigation approach:\n\n**Primary bottom nav items (always visible):**\n1. Home/Dashboard - most frequent access point\n2. Messages - high-priority feature requiring quick access\n3. Search - universal need across app contexts\n4. Profile/More - gateway to secondary features\n\n**Implementation details:**\n- Use Material Design bottom navigation on Android\n- Use UITabBar on iOS with platform-appropriate styling\n- Overflow menu (\"More\" tab) contains Settings, Help, and other secondary features\n- Bottom nav auto-hides on scroll down, reappears on scroll up to maximize content space\n- Support for iPad/tablet: transform to side rail navigation on larger screens\n\n**Validation results:**\n- 34% improvement in task completion time for core user journeys\n- 89% user preference in A/B testing vs. drawer navigation\n- Meets WCAG 2.1 AA accessibility standards with VoiceOver/TalkBack\n\n**Trade-offs accepted:**\n- Limited to 4-5 primary navigation items (acceptable given our feature prioritization)\n- Requires thoughtful overflow menu organization (mitigated with smart ordering by usage frequency)\n\nThis approach provides the best balance of discoverability, one-handed usability, and platform consistency."
}')
echo "   ✓ Created conclusion: Adaptive bottom navigation"

# Create a follow-up question
record3=$(api_call "create_record" '{
    "parent_id": "'$record1'",
    "type": "question",
    "title": "How do we handle notification badges on the bottom nav?",
    "summary": "Determining badge behavior for Messages and other nav items with new content",
    "body": "With the new bottom navigation established, we need to define notification badge behavior:\n\n**Current state:**\n- Messages shows numeric badge (1-99+)\n- No other nav items show badges\n- Push notifications handled separately\n\n**Design questions:**\n1. Should we show badges on other tabs (e.g., Dashboard for new items)?\n2. Numeric vs. dot indicators?\n3. How do badges interact with the auto-hide navigation behavior?\n4. Badge persistence - when do they clear?\n5. Accessibility announcements for badge updates\n\n**User feedback:**\n- Some users report missing important updates because they don'\''t check all tabs\n- Others find too many badges \"noisy\" and ignore them\n- Accessibility users need clear badge semantics\n\nWe need to establish a consistent badge strategy that improves awareness without overwhelming users."
}')
echo "   ✓ Created follow-up question: Notification badges"

# Create another root-level question
record4=$(api_call "create_record" '{
    "parent_id": null,
    "type": "question",
    "title": "What color system should we adopt for the redesign?",
    "summary": "Evaluating color palette options considering accessibility, brand, and dark mode support",
    "body": "The current color system has grown organically and lacks consistency. For the redesign, we need a systematic approach to color.\n\n**Current problems:**\n- 23 different shades of blue used inconsistently\n- Poor contrast ratios in several UI elements (WCAG AA failures)\n- Dark mode is an afterthought with manual color overrides\n- No semantic color tokens (colors are referenced by literal values)\n\n**Requirements:**\n- WCAG 2.1 AA compliance minimum (AAA for critical text)\n- Native dark mode support\n- Support for user-customizable accent colors\n- Colorblind-friendly (avoid red/green only distinctions)\n- Clear semantic naming (primary, danger, success, etc.)\n\n**Options being considered:**\n1. Material Design 3 color system with dynamic theming\n2. Custom palette with HSL-based systematic generation\n3. Radix Colors (predetermined accessible palettes)\n\nEach option has different implications for design workflow, engineering implementation, and long-term maintenance."
}')
echo "   ✓ Created root question: Color system"

# Create a deferred/LATER item
record5=$(api_call "create_record" '{
    "parent_id": "'$record4'",
    "type": "note",
    "title": "Research: Color contrast verification tooling",
    "summary": "Evaluate automated tools for continuous color contrast validation in CI/CD",
    "state": "LATER",
    "body": "Once we establish the color system, we should integrate automated contrast checking into our development workflow.\n\n**Tools to evaluate:**\n- Pa11y for automated accessibility testing\n- axe-core for component-level validation\n- Chromatic for visual regression with accessibility checks\n- Custom Figma plugins for design-time validation\n\n**Goals:**\n- Prevent contrast ratio regressions in PRs\n- Validate all color combinations automatically\n- Integrate with Figma design workflow\n- Generate accessibility reports for stakeholders\n\nThis is important but not blocking the initial color system decision. Marking as LATER to revisit after core color palette is established."
}')
echo "   ✓ Created deferred note: Contrast verification tooling"

# Create a second project
echo ""
echo "📁 Creating second sample project..."
project2_response=$(api_call "create_project" '{
    "id": "api-architecture",
    "name": "API Architecture Evolution",
    "description": "Planning the migration from REST to GraphQL for improved client flexibility"
}')
echo "   Created: API Architecture Evolution"
echo ""

# Create records in second project - need to activate session first
record6=$(api_call "create_record" '{
    "parent_id": null,
    "type": "question",
    "title": "Should we migrate to GraphQL or improve our REST API?",
    "summary": "Weighing the benefits of GraphQL adoption vs. REST API enhancements",
    "body": "Our mobile and web clients currently consume a REST API that has grown to over 100 endpoints. We'\''re experiencing several pain points:\n\n**Current REST API issues:**\n- Over-fetching: Mobile clients download unnecessary data, impacting performance\n- Under-fetching: Multiple round trips required for complex views (N+1 problems)\n- Version management: Supporting multiple API versions is becoming complex\n- Documentation drift: OpenAPI specs don'\''t always match implementation\n- Client-specific endpoints: We'\''ve created special endpoints for specific client needs\n\n**GraphQL potential benefits:**\n- Clients request exactly the data they need\n- Single endpoint with schema-driven development\n- Strong typing and introspection\n- Better developer experience with GraphQL Playground\n- Reduced version management complexity\n\n**GraphQL concerns:**\n- Learning curve for team (backend and frontend)\n- Query complexity and performance (need query cost analysis)\n- Caching strategy differs from REST\n- Migration path from existing REST API\n- Potential for abusive queries without proper safeguards\n\n**Alternative: REST API improvements:**\n- Implement JSON:API or similar standards for consistency\n- Add sparse fieldsets (field filtering) to reduce over-fetching\n- Improve batching with composite endpoints\n- Better documentation with updated OpenAPI specs\n- Maintain existing client compatibility\n\nWe need to make an informed decision based on our specific needs, team capabilities, and long-term maintenance implications."
}')
echo "   ✓ Created root question: GraphQL vs REST"

echo ""
echo "✅ Sample data generation complete!"
echo ""
echo "📊 Summary:"
echo "   • 2 projects created"
echo "   • 6 records created across projects"
echo "   • Mix of questions, conclusions, and notes"
echo "   • Includes OPEN and LATER states"
echo "   • Records contain 1-3 paragraph bodies (realistic length)"
echo ""
echo "🔍 Try these commands to explore the data:"
echo ""
echo "List all projects:"
echo "  curl -X POST $BASE_URL \\"
echo "    -H 'Content-Type: application/json' \\"
if [ "$use_auth" = true ]; then
    echo "    -H 'Authorization: Bearer $API_KEY' \\"
fi
echo "    -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"list_projects\",\"params\":{}}'"
echo ""
echo "Get project overview:"
echo "  curl -X POST $BASE_URL \\"
echo "    -H 'Content-Type: application/json' \\"
if [ "$use_auth" = true ]; then
    echo "    -H 'Authorization: Bearer $API_KEY' \\"
fi
echo "    -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"get_project_overview\",\"params\":{}}'"
echo ""
echo "Search records:"
echo "  curl -X POST $BASE_URL \\"
echo "    -H 'Content-Type: application/json' \\"
if [ "$use_auth" = true ]; then
    echo "    -H 'Authorization: Bearer $API_KEY' \\"
fi
echo "    -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"search_records\",\"params\":{\"query\":\"navigation\"}}'"
echo ""
