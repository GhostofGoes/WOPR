/*
  Picks the install tabs (the install-tabs shortcode) for the reader's system: Windows, macOS or
  Linux among the download tabs and, among the command-line tabs, Linux (apt) or Linux (RPM) when
  the browser names a distribution that uses them, and Linux (any) otherwise. It leaves alone a set
  of tabs the reader has picked before, which Hextra saved, and it saves nothing itself, so the
  reader's own click always wins. Phones and tablets, ChromeOS and anything else unknown keep the
  first tab, as they do without JavaScript.

  Each download tab has two buttons, one for Intel and AMD computers and one for Arm computers. When
  the browser says for certain which processor the computer has, it hides the button that does not
  fit; when it cannot tell, both stay, as they do without JavaScript. Chrome and Edge say it in the
  client hints they give when asked, on every system. Without client hints, only Firefox on Linux
  names the processor in navigator.platform and its user agent: Windows and macOS browsers name
  x64 or Intel even on Arm computers, and Chrome's user agent says x86_64 on every Linux.

  layouts/_partials/custom/head-end.html loads it, deferred, on the pages with install tabs. It
  runs before Hextra's own tabs script, which restores the saved picks and handles clicks.
*/
(function () {
  'use strict';

  // Hextra's updateGroup(container, index), copied from its js/core/tabs.js when the site is built
  // (head-end.html puts it in place of the next line). So a tab picked here is drawn and marked for
  // screen readers exactly as one the reader clicked: data-state, aria-selected and tabindex on the
  // tabs, aria-hidden on the panels. A Hextra that changes the function fails the build, instead
  // of leaving this script to drift from it.
  /* @HEXTRA-UPDATE-GROUP@ */

  var ua = navigator.userAgent || '';
  var uaData = navigator.userAgentData;

  // 'windows', 'macos', 'linux', or '' when the tabs should stay as they are.
  function system() {
    var platform = (uaData && uaData.platform) || navigator.platform || '';
    // ChromeOS's user agent says CrOS. codespell:ignore cros
    if ((uaData && uaData.mobile) || /Android|iPhone|iPad|iPod|CrOS/.test(ua)) { // codespell:ignore cros
      return '';
    }
    if (/^Win/i.test(platform)) {
      return 'windows';
    }
    if (/^Mac/i.test(platform)) {
      // An iPad asks for the desktop site as a Mac, but has a touch screen.
      return navigator.maxTouchPoints > 1 ? '' : 'macos';
    }
    if (/Linux/i.test(platform) && !/Android/i.test(platform)) {
      return 'linux';
    }
    return '';
  }

  // The tab names to look for, best first. Each set of tabs takes the first one it has.
  function wanted(sys) {
    if (sys === 'windows') {
      return ['Windows'];
    }
    if (sys === 'macos') {
      return ['macOS'];
    }
    if (/Ubuntu|Debian|Mint|Pop!_OS|elementary/i.test(ua)) {
      return ['Linux (apt)', 'Linux'];
    }
    if (/Fedora|Red Hat|CentOS|Rocky|AlmaLinux|SUSE/i.test(ua)) {
      return ['Linux (RPM)', 'Linux'];
    }
    return ['Linux (any)', 'Linux'];
  }

  // Whether the reader has picked a tab in this set before: Hextra saves the pick under the set's
  // tab names, joined by commas.
  function picked(names) {
    try {
      return localStorage.getItem('hextra-tab-' + encodeURIComponent(names)) !== null;
    } catch (e) {
      return false;
    }
  }

  function pickTabs(sys) {
    var want = wanted(sys);
    var sets = {};
    document.querySelectorAll('.wopr-install-tabs [data-tab-group]').forEach(function (el) {
      sets[el.dataset.tabGroup] = true;
    });
    // Every set with the same names, such as the Uninstall section's, is synced with them.
    document.querySelectorAll('[data-tab-group]').forEach(function (group) {
      var names = group.dataset.tabGroup;
      if (!sets[names] || picked(names)) {
        return;
      }
      var tabs = names.split(',');
      for (var i = 0; i < want.length; i++) {
        var index = tabs.indexOf(want[i]);
        if (index >= 0) {
          updateGroup(group, index);
          return;
        }
      }
    });
  }

  // Shows only the buttons for arch, 'x86-64' or 'arm64'.
  function showArch(arch) {
    document.querySelectorAll('.wopr-download').forEach(function (pair) {
      var buttons = pair.querySelectorAll('[data-wopr-arch]');
      var fits = pair.querySelectorAll('[data-wopr-arch="' + arch + '"]');
      // A pair without a button for arch keeps both.
      if (fits.length === 0) {
        return;
      }
      buttons.forEach(function (el) {
        el.hidden = el.dataset.woprArch !== arch;
      });
    });
  }

  function pickArch(sys) {
    if (uaData && uaData.getHighEntropyValues) {
      uaData
        .getHighEntropyValues(['architecture', 'bitness'])
        .then(function (v) {
          if (v.bitness !== '64') {
            return;
          }
          if (v.architecture === 'arm') {
            showArch('arm64');
          } else if (v.architecture === 'x86') {
            showArch('x86-64');
          }
        })
        .catch(function () {});
      return;
    }
    if (sys !== 'linux') {
      return;
    }
    var names = navigator.platform + ' ' + ua;
    var arm = /aarch64|arm64/i.test(names);
    var x86 = /x86_64|amd64/i.test(names);
    if (arm !== x86) {
      showArch(arm ? 'arm64' : 'x86-64');
    }
  }

  var sys = system();
  if (sys) {
    pickTabs(sys);
    pickArch(sys);
  }
})();
