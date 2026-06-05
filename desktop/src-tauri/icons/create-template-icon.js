const fs = require('fs');

// Create a template icon that shows the Venn diagram structure clearly
// Using filled regions to show the overlapping pattern

const svgTemplate = `<svg width="1024" height="1024" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg">
  <!-- Zoo Logo: Template Version for macOS Menu Bar -->
  <!-- Uses filled regions to show the Venn diagram structure -->

  <defs>
    <clipPath id="outerCircle">
      <circle cx="512" cy="508" r="259"/>
    </clipPath>
    <clipPath id="greenClip">
      <circle cx="515" cy="368" r="227"/>
    </clipPath>
    <clipPath id="redClip">
      <circle cx="369" cy="593" r="227"/>
    </clipPath>
    <clipPath id="blueClip">
      <circle cx="653" cy="594" r="227"/>
    </clipPath>

    <!-- Create masks for each unique region -->
    <mask id="greenOnly">
      <rect width="1024" height="1024" fill="white"/>
      <circle cx="369" cy="593" r="227" fill="black"/>
      <circle cx="653" cy="594" r="227" fill="black"/>
    </mask>

    <mask id="redOnly">
      <rect width="1024" height="1024" fill="white"/>
      <circle cx="515" cy="368" r="227" fill="black"/>
      <circle cx="653" cy="594" r="227" fill="black"/>
    </mask>

    <mask id="blueOnly">
      <rect width="1024" height="1024" fill="white"/>
      <circle cx="515" cy="368" r="227" fill="black"/>
      <circle cx="369" cy="593" r="227" fill="black"/>
    </mask>
  </defs>

  <g clip-path="url(#outerCircle)">
    <!-- Base circles with low opacity -->
    <circle cx="515" cy="368" r="227" fill="black" opacity="0.15"/>
    <circle cx="369" cy="593" r="227" fill="black" opacity="0.15"/>
    <circle cx="653" cy="594" r="227" fill="black" opacity="0.15"/>

    <!-- Two-way overlaps with higher opacity -->
    <g clip-path="url(#greenClip)">
      <circle cx="369" cy="593" r="227" fill="black" opacity="0.25"/>
    </g>

    <g clip-path="url(#greenClip)">
      <circle cx="653" cy="594" r="227" fill="black" opacity="0.25"/>
    </g>

    <g clip-path="url(#redClip)">
      <circle cx="653" cy="594" r="227" fill="black" opacity="0.25"/>
    </g>

    <!-- Center overlap with highest opacity -->
    <g clip-path="url(#greenClip)">
      <g clip-path="url(#redClip)">
        <circle cx="653" cy="594" r="227" fill="black" opacity="0.35"/>
      </g>
    </g>
  </g>
</svg>`;

// Save the template SVG
fs.writeFileSync('zoo-template-icon.svg', svgTemplate);

// Create a simpler version with just outlines and dots
const outlineTemplate = `<svg width="1024" height="1024" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg">
  <!-- Zoo Logo: Outline Template for macOS Menu Bar -->

  <defs>
    <clipPath id="outerCircle">
      <circle cx="512" cy="508" r="259"/>
    </clipPath>
  </defs>

  <g clip-path="url(#outerCircle)">
    <!-- Three circle outlines with thicker strokes -->
    <circle cx="515" cy="368" r="227" fill="none" stroke="black" stroke-width="20"/>
    <circle cx="369" cy="593" r="227" fill="none" stroke="black" stroke-width="20"/>
    <circle cx="653" cy="594" r="227" fill="none" stroke="black" stroke-width="20"/>

    <!-- Add dots at circle centers for visibility -->
    <circle cx="515" cy="368" r="15" fill="black"/>
    <circle cx="369" cy="593" r="15" fill="black"/>
    <circle cx="653" cy="594" r="15" fill="black"/>

    <!-- Center dot where all three overlap -->
    <circle cx="512" cy="508" r="20" fill="black"/>
  </g>
</svg>`;

fs.writeFileSync('zoo-template-outline.svg', outlineTemplate);

// Create a hybrid version with both fills and outlines
const hybridTemplate = `<svg width="1024" height="1024" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg">
  <!-- Zoo Logo: Hybrid Template for macOS Menu Bar -->

  <defs>
    <clipPath id="outerCircle">
      <circle cx="512" cy="508" r="259"/>
    </clipPath>
    <clipPath id="greenClip">
      <circle cx="515" cy="368" r="227"/>
    </clipPath>
    <clipPath id="redClip">
      <circle cx="369" cy="593" r="227"/>
    </clipPath>
    <clipPath id="blueClip">
      <circle cx="653" cy="594" r="227"/>
    </clipPath>
  </defs>

  <g clip-path="url(#outerCircle)">
    <!-- Light fills to show regions -->
    <circle cx="515" cy="368" r="227" fill="black" opacity="0.08"/>
    <circle cx="369" cy="593" r="227" fill="black" opacity="0.08"/>
    <circle cx="653" cy="594" r="227" fill="black" opacity="0.08"/>

    <!-- Strong outlines for definition -->
    <circle cx="515" cy="368" r="227" fill="none" stroke="black" stroke-width="8" opacity="0.8"/>
    <circle cx="369" cy="593" r="227" fill="none" stroke="black" stroke-width="8" opacity="0.8"/>
    <circle cx="653" cy="594" r="227" fill="none" stroke="black" stroke-width="8" opacity="0.8"/>

    <!-- Emphasize overlaps -->
    <g clip-path="url(#greenClip)">
      <circle cx="369" cy="593" r="227" fill="black" opacity="0.1"/>
    </g>

    <g clip-path="url(#greenClip)">
      <circle cx="653" cy="594" r="227" fill="black" opacity="0.1"/>
    </g>

    <g clip-path="url(#redClip)">
      <circle cx="653" cy="594" r="227" fill="black" opacity="0.1"/>
    </g>

    <!-- Center emphasis -->
    <g clip-path="url(#greenClip)">
      <g clip-path="url(#redClip)">
        <circle cx="653" cy="594" r="227" fill="black" opacity="0.15"/>
      </g>
    </g>
  </g>
</svg>`;

fs.writeFileSync('zoo-template-hybrid.svg', hybridTemplate);

console.log('Created three template variations:');
console.log('1. zoo-template-icon.svg - Filled regions with opacity layers');
console.log('2. zoo-template-outline.svg - Outlines with center dots');
console.log('3. zoo-template-hybrid.svg - Combination of fills and outlines');
console.log('\nNow converting to PNG icons...');