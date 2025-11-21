# NeuroNetes Project Website

This directory contains the project website for NeuroNetes. The website showcases features, examples, and performance benchmarks.

## Structure

- `index.html` - Main landing page with feature overview
- `examples.html` - Detailed examples with configurations and expected results
- `benchmarks.html` - Performance benchmarks and real-world results

## Pages

### Home (index.html)
- Project overview
- Key features grid
- Architecture diagram
- Quick start guide
- Documentation links

### Examples (examples.html)
- **Chat Agent** - Simple conversational AI with complete configuration
- **Code Assistant** - RAG-powered code helper with tools
- **RAG Pipeline** - Complete retrieval-augmented generation setup
- Comparison tables
- Client library examples
- Expected performance results

### Benchmarks (benchmarks.html)
- Latency benchmarks (TTFT)
- Throughput metrics
- Autoscaling performance
- GPU efficiency
- Cost optimization results
- Test coverage statistics

## Deployment

### GitHub Pages

To deploy to GitHub Pages:

1. Enable GitHub Pages in repository settings
2. Set source to `/docs/website` directory
3. Website will be available at `https://bowenislandsong.github.io/NeuroNetes/website/`

Alternatively, use the root `docs/` directory:

```bash
# Move website files to docs root for GitHub Pages
cp -r docs/website/* docs/
```

### Local Preview

Simply open the HTML files in a browser:

```bash
# Open in default browser (macOS)
open docs/website/index.html

# Or use a simple HTTP server
cd docs/website
python3 -m http.server 8000
# Visit http://localhost:8000
```

## Features

- **Responsive Design** - Works on desktop, tablet, and mobile
- **Modern UI** - Clean, professional design with gradient accents
- **Interactive Elements** - Hover effects, smooth scrolling
- **Code Examples** - Syntax-highlighted configuration examples
- **Performance Metrics** - Visual charts and comparison tables
- **No Dependencies** - Pure HTML/CSS, no build process required

## Customization

### Colors

The website uses a consistent color scheme:
- Primary: `#667eea` (purple-blue gradient)
- Secondary: `#764ba2` (purple)
- Success: `#48bb78` (green)
- Warning: `#ed8936` (orange)
- Dark: `#2d3748` (navy)

### Adding New Pages

1. Copy an existing HTML file
2. Update the `<title>` tag
3. Update navigation links
4. Add content following existing patterns
5. Link from other pages

### Updating Metrics

Update the performance numbers in `benchmarks.html`:
- Locate the metric in the appropriate section
- Update the `.number` or `.value` divs
- Update comparison tables as needed

## Best Practices

- Keep page load fast (no large images or external dependencies)
- Maintain consistent styling across pages
- Update all navigation links when adding pages
- Test on multiple browsers
- Optimize for both desktop and mobile viewports

## Contributing

When adding new content:

1. Follow the existing HTML structure
2. Use semantic HTML elements
3. Maintain accessibility (alt text, ARIA labels)
4. Test responsive behavior
5. Keep code examples accurate and up-to-date

## License

Apache License 2.0 - Same as the main NeuroNetes project
