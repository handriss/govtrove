(function () {
  if (localStorage.getItem('ph_opt_out') === '1' ||
      window.location.hostname === 'localhost') return;

  var script = document.createElement('script');
  script.src = 'https://us-assets.i.posthog.com/static/array.js';
  script.onload = function () {
    posthog.init('phc_sQkqcZGsl759FtVCLHZpgMyFQ6DaWrsJR3VysfCVO8Z', {
      api_host: 'https://k.govtrove.com',
      autocapture: false,
      capture_pageview: true,
      capture_pageleave: true,
      persistence: 'memory',
      person_profiles: 'identified_only',
      disable_toolbar: true,
    });

    var params = new URLSearchParams(window.location.search);
    var props = {};
    if (params.get('utm_source')) props.utm_source = params.get('utm_source');
    if (params.get('utm_medium')) props.utm_medium = params.get('utm_medium');
    if (params.get('utm_campaign')) props.utm_campaign = params.get('utm_campaign');
    if (Object.keys(props).length > 0) posthog.register(props);
  };
  document.head.appendChild(script);
})();
