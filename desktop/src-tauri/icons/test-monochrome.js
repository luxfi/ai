const fs = require('fs');
const { chromium } = require('playwright');

async function testMonochrome() {
    const browser = await chromium.launch({ headless: false });
    const context = await browser.newContext();
    const page = await context.newPage();

    // Load the adjuster page
    await page.goto(`file://${__dirname}/final-adjuster.html`);

    // Switch to monochrome mode
    await page.click('#monoTab');

    // Wait for rendering
    await page.waitForTimeout(500);

    // Try different stroke widths to find what's visible
    console.log('Testing different stroke widths...');

    for (let width = 4; width <= 24; width += 4) {
        await page.fill('#strokeWidth', width.toString());
        await page.evaluate((w) => {
            document.getElementById('strokeWidth').value = w;
            document.getElementById('strokeWidth').dispatchEvent(new Event('input'));
            updateSVG();
        }, width);

        await page.waitForTimeout(200);

        // Take a screenshot to see what's rendering
        await page.screenshot({
            path: `mono-test-stroke-${width}.png`,
            clip: { x: 270, y: 100, width: 400, height: 400 }
        });

        console.log(`Tested stroke width: ${width}`);
    }

    // Test with different approaches
    console.log('\nTrying alternative rendering approach...');

    // Override the SVG generation to test different approaches
    await page.evaluate(() => {
        window.updateSVG = function() {
            const outerRadius = 259;
            const outerX = 512;
            const outerY = 508;
            const circleRadius = 227;
            const greenX = 515;
            const greenY = 368;
            const redX = 369;
            const redY = 593;
            const blueX = 653;
            const blueY = 594;
            const strokeWidth = parseInt(document.getElementById('strokeWidth').value);

            // Try a different approach - filled shapes with mask
            const svgContent = `
                <defs>
                    <mask id="vennMask">
                        <rect width="1024" height="1024" fill="white"/>
                        <circle cx="${greenX}" cy="${greenY}" r="${circleRadius - strokeWidth}" fill="black"/>
                        <circle cx="${redX}" cy="${redY}" r="${circleRadius - strokeWidth}" fill="black"/>
                        <circle cx="${blueX}" cy="${blueY}" r="${circleRadius - strokeWidth}" fill="black"/>
                    </mask>
                    <clipPath id="outerClip">
                        <circle cx="${outerX}" cy="${outerY}" r="${outerRadius}"/>
                    </clipPath>
                </defs>

                <g clip-path="url(#outerClip)">
                    <!-- Filled circles with mask to create outlines -->
                    <g mask="url(#vennMask)">
                        <circle cx="${greenX}" cy="${greenY}" r="${circleRadius}" fill="black"/>
                        <circle cx="${redX}" cy="${redY}" r="${circleRadius}" fill="black"/>
                        <circle cx="${blueX}" cy="${blueY}" r="${circleRadius}" fill="black"/>
                    </g>

                    <!-- Outer ring -->
                    <circle cx="${outerX}" cy="${outerY}" r="${outerRadius}" fill="none" stroke="black" stroke-width="${strokeWidth}"/>
                </g>
            `;

            document.getElementById('svgCanvas').innerHTML = svgContent;
            document.getElementById('overlaySVG').innerHTML = svgContent;
        };

        updateSVG();
    });

    await page.waitForTimeout(500);
    await page.screenshot({
        path: 'mono-test-mask-approach.png',
        clip: { x: 270, y: 100, width: 400, height: 400 }
    });

    // Try pure fill approach for visibility
    await page.evaluate(() => {
        window.updateSVG = function() {
            const outerRadius = 259;
            const outerX = 512;
            const outerY = 508;
            const circleRadius = 227;
            const greenX = 515;
            const greenY = 368;
            const redX = 369;
            const redY = 593;
            const blueX = 653;
            const blueY = 594;

            // Simple filled version to ensure visibility
            const svgContent = `
                <defs>
                    <clipPath id="outerCircle">
                        <circle cx="${outerX}" cy="${outerY}" r="${outerRadius}"/>
                    </clipPath>
                </defs>

                <g clip-path="url(#outerCircle)">
                    <!-- Background -->
                    <rect width="1024" height="1024" fill="white"/>

                    <!-- Three circles as solid black -->
                    <circle cx="${greenX}" cy="${greenY}" r="${circleRadius}" fill="black" opacity="0.2"/>
                    <circle cx="${redX}" cy="${redY}" r="${circleRadius}" fill="black" opacity="0.2"/>
                    <circle cx="${blueX}" cy="${blueY}" r="${circleRadius}" fill="black" opacity="0.2"/>
                </g>
            `;

            document.getElementById('svgCanvas').innerHTML = svgContent;
        };

        updateSVG();
    });

    await page.waitForTimeout(500);
    await page.screenshot({
        path: 'mono-test-filled.png',
        clip: { x: 270, y: 100, width: 400, height: 400 }
    });

    console.log('Screenshots saved. Check mono-test-*.png files');

    // Keep browser open for manual testing
    console.log('\nBrowser will stay open for manual testing...');
    console.log('Try adjusting the stroke width slider and observe the changes');

    // Wait for a long time to keep browser open
    await page.waitForTimeout(60000);

    await browser.close();
}

testMonochrome().catch(console.error);