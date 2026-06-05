const path = require('path');
const { chromium } = require('playwright');

(async () => {
    const browser = await chromium.launch({ headless: false });
    const page = await browser.newPage();

    // Load the adjuster tool
    const filePath = 'file://' + path.resolve(__dirname, 'final-adjuster.html');
    await page.goto(filePath);

    // Set the values from your screenshot
    // Outer Circle: 259 radius, X: 515, Y: 515
    // Circle Size: 224
    // Green: X: 515, Y: 373
    // Red: X: 376, Y: 595
    // Blue: X: 658, Y: 596

    await page.fill('#outerRadius', '259');
    await page.fill('#outerX', '515');
    await page.fill('#outerY', '515');
    await page.fill('#circleRadius', '224');

    // Set slider values
    await page.evaluate(() => {
        document.getElementById('greenX').value = 515;
        document.getElementById('greenY').value = 373;
        document.getElementById('redX').value = 376;
        document.getElementById('redY').value = 595;
        document.getElementById('blueX').value = 658;
        document.getElementById('blueY').value = 596;

        // Trigger updates
        ['greenX', 'greenY', 'redX', 'redY', 'blueX', 'blueY'].forEach(id => {
            const event = new Event('input', { bubbles: true });
            document.getElementById(id).dispatchEvent(event);
        });
    });

    console.log('Adjuster loaded with your values');
    console.log('Press Ctrl+C to exit');

    // Keep browser open
    await new Promise(() => {});
})();