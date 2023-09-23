import express from "express";

import { NavigationBar } from "../static/js/components/navigation-bar.js";
import "../static/js/components/home-page.js";

import {render} from '@lit-labs/ssr';
// import {RenderResultReadable} from '@lit-labs/ssr/lib/render-result-readable.js';
import {collectResult, collectResultSync} from '@lit-labs/ssr/lib/render-result.js';
import { html } from "lit";

const app = express();
app.use('/public', express.static('static'))

// Set up the navigation bar middleware
app.use((_, res, next) => {
    const navigationBar = new NavigationBar();
    const renderedNavigationBar = render(navigationBar.render());
  
    res.locals.navigationBar = collectResultSync(renderedNavigationBar);
  
    next();
  });
  

// Render the home page
app.get("/", async (_, res) => {
    // const homePage = new HomePage();
    const result = render(html`<home-page></home-page>`);
    const renderedHtml = await collectResult(result);
    // const renderedHomePage = homePage.render();
    // res.setHeader('Content-Type', 'text/html');
    // res.write('<!DOCTYPE html>');
    // res.write('<html>');
    // res.write('<head>');
    // res.write('<title>My App</title>');
    // res.write('</head>');
    // res.write('<body>');
    // res.write(renderedHtml);
    // res.write('<script src="/public/index.js"></script>');
    // res.write('</body>');
    // res.write('</html>');
    res.write(`
      <!DOCTYPE html>
      <html>
        <head>
          <!-- On browsers that don't yet support native declarative shadow DOM, a
              paint can occur after some or all pre-rendered HTML has been parsed,
              but before the declarative shadow DOM polyfill has taken effect. This
              paint is undesirable because it won't include any component shadow DOM.
              To prevent layout shifts that can result from this render, we use a
              "dsd-pending" attribute to ensure we only paint after we know
              shadow DOM is active. -->
          <style>
            body[dsd-pending] {
              display: none;
            }
          </style>
          <title>Home Page</title>
        </head>
        <body dsd-pending>
          <script>
            if (HTMLTemplateElement.prototype.hasOwnProperty('shadowRoot')) {
              // This browser has native declarative shadow DOM support, so we can
              // allow painting immediately.
              document.body.removeAttribute('dsd-pending');
            }
          </script>
          <script src="/public/index.js"></script>
          ${res.locals.navigationBar}
          <main>${renderedHtml}</main>
          <br />
          Hi
          <script type="module">
            // Check if we require the template shadow root polyfill.
            if (!HTMLTemplateElement.prototype.hasOwnProperty('shadowRoot')) {
              // Fetch the template shadow root polyfill.
              const {hydrateShadowRoots} = await import(
                '/node_modules/@webcomponents/template-shadowroot/template-shadowroot.js'
              );

              // Apply the polyfill. This is a one-shot operation, so it is important
              // it happens after all HTML has been parsed.
              hydrateShadowRoots(document.body);

              // At this point, browsers without native declarative shadow DOM
              // support can paint the initial state of your components!
              document.body.removeAttribute('dsd-pending');
            }
          </script>

        </body>
      </html>
    `);
    // res.end();
  });
  
//   // Render the item details page
//   app.get("/items/:itemId", async (req, res) => {
//     const itemId = req.params.itemId;
  
//     const itemDetailsPage = new ItemDetailsPage();
//     itemDetailsPage.setAttribute("item-id", itemId);
  
//     const renderedItemDetailsPage = itemDetailsPage.renderToString();
  
//     res.send(`
//       <!DOCTYPE html>
//       <html>
//         <head>
//           <title>Item Details</title>
//         </head>
//         <body>
//           ${res.locals.navigationBar}
//           <main>${renderedItemDetailsPage}</main>
//         </body>
//       </html>
//     `);
//   });

app.listen(3002, () => {
    console.log("Server listening on port 3002");
});