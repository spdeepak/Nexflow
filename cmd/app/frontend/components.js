// --- Reusable UI components ---

// Inline SVG icon helpers (fill/stroke driven by currentColor so they inherit
// the button/link color and hover styles).

const ICON_PATHS = {
    bullet: {
        viewBox: '0 0 16 16',
        paths: [
            'M8 16C3.58172 16 0 12.4183 0 8C0 3.58172 3.58172 0 8 0C12.4183 0 16 3.58172 16 8C16 12.4183 12.4183 16 8 16z',
        ],
        fill: true,
    },
    back: {
        viewBox: '0 0 24 24',
        paths: [
            'M19 12H5M5 12L12 19M5 12L12 5',
        ],
        stroke: true,
    },
};

function iconSvg(name, className = '') {
    const def = ICON_PATHS[name];
    if (!def) return '';
    const common = `viewBox="${def.viewBox}" xmlns="http://www.w3.org/2000/svg" aria-hidden="true"`;
    const stroke = def.stroke ? ' fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"' : '';
    const fill = def.fill ? ' fill="currentColor"' : '';
    const paths = def.paths
        .map(p => def.fill
            ? `<path fill-rule="evenodd" clip-rule="evenodd" d="${p}"/>`
            : `<path d="${p}"/>`)
        .join('\n    ');
    return `<svg class="${className}" ${common}${stroke}${fill}>\n    ${paths}\n</svg>`;
}