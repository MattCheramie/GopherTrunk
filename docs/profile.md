---
layout: page
title: Edit profile
description: Edit your GopherTrunk profile — avatar, display name, amateur radio call sign, location, bio, and links. Shown on your public profile page.
permalink: /profile/
nav_group: Community
hide_ctas: true
---

# Edit profile

This is what other operators see on your **[public profile](/u/)**. Everything is
optional — fill in as much or as little as you like.

<div class="profile-edit" data-profile-edit>
  <p class="account__loading" data-loading>Loading…</p>

  <section class="profile-edit__signedout" data-signedout hidden>
    <p>You need to <a href="/account/">sign in</a> to edit your profile.</p>
  </section>

  <form class="account__form profile-edit__form" data-form hidden novalidate>
    <div class="profile-edit__avatar">
      <img class="profile-edit__avatar-img" data-avatar-img alt="" width="96" height="96">
      <div class="profile-edit__avatar-controls">
        <label class="profile-edit__avatar-label">
          Profile picture
          <input type="file" name="avatar" accept="image/png,image/jpeg,image/webp" data-avatar-input>
        </label>
        <p class="account__msg" data-avatar-msg role="status" aria-live="polite"></p>
        <p class="profile-edit__hint">PNG, JPEG, or WebP, up to 2&nbsp;MB.</p>
      </div>
    </div>

    <label>Display name
      <input type="text" name="display_name" maxlength="60" autocomplete="name">
    </label>
    <label>Username
      <input type="text" name="username" maxlength="30" autocomplete="off" spellcheck="false">
      <span class="profile-edit__hint">Letters, numbers, <code>_</code> and <code>-</code>. Your public profile lives at <code>/u/?u=&lt;username&gt;</code>.</span>
    </label>
    <label>Amateur radio call sign
      <input type="text" name="callsign" maxlength="10" autocomplete="off" spellcheck="false" placeholder="e.g. W1AW">
      <span class="profile-edit__hint">Optional. Shown as a badge on your profile.</span>
    </label>

    <fieldset class="profile-edit__location">
      <legend>Location <span class="profile-edit__hint">(optional)</span></legend>
      <label>Country
        <select name="country" data-country>
          <option value="">— Not set —</option>
        </select>
      </label>
      <label>State / region
        <input type="text" name="region" maxlength="80" autocomplete="address-level1">
      </label>
      <label>City
        <input type="text" name="city" maxlength="80" autocomplete="address-level2">
      </label>
    </fieldset>

    <label>Bio
      <textarea name="bio" rows="4" maxlength="1000" placeholder="A short intro — rigs you run, systems you follow, what you’re decoding."></textarea>
    </label>
    <label>Website
      <input type="url" name="website" maxlength="200" autocomplete="url" placeholder="https://…">
    </label>

    <p class="profile-edit__since" data-since hidden>Member since <span data-since-val></span>.</p>

    <div class="account__form-actions">
      <button type="submit" class="btn">Save profile</button>
      <a class="btn btn--secondary" data-view-link href="/u/">View public profile</a>
    </div>
    <p class="account__msg" data-msg role="status" aria-live="polite"></p>
  </form>
</div>

<script>
document.addEventListener('DOMContentLoaded', function () {
  'use strict';
  var root = document.querySelector('[data-profile-edit]');
  if (!root) return;
  var loading   = root.querySelector('[data-loading]');
  var signedout = root.querySelector('[data-signedout]');
  var form      = root.querySelector('[data-form]');
  var msg       = root.querySelector('[data-msg]');
  var avatarImg = root.querySelector('[data-avatar-img]');
  var avatarIn  = root.querySelector('[data-avatar-input]');
  var avatarMsg = root.querySelector('[data-avatar-msg]');
  var countrySel = root.querySelector('[data-country]');
  var sinceBox  = root.querySelector('[data-since]');
  var viewLink  = root.querySelector('[data-view-link]');

  function show(el) { if (el) el.hidden = false; }
  function hide(el) { if (el) el.hidden = true; }
  function setMsg(box, text, ok) {
    if (!box) return;
    box.textContent = text || '';
    box.className = 'account__msg' + (ok === false ? ' is-error' : ok === true ? ' is-ok' : '');
  }
  function field(name) { return form.querySelector('[name=' + name + ']'); }
  var DEFAULT_AVATAR = '/assets/gophertrunk-logo.png';

  // ISO-3166 alpha-2 country list, built into the <select> in code (keeps the
  // markup clean; the data is ours, so it's safe to set via option text).
  var COUNTRIES = [
    ['AF','Afghanistan'],['AX','Åland Islands'],['AL','Albania'],['DZ','Algeria'],['AS','American Samoa'],['AD','Andorra'],['AO','Angola'],['AI','Anguilla'],['AQ','Antarctica'],['AG','Antigua and Barbuda'],['AR','Argentina'],['AM','Armenia'],['AW','Aruba'],['AU','Australia'],['AT','Austria'],['AZ','Azerbaijan'],['BS','Bahamas'],['BH','Bahrain'],['BD','Bangladesh'],['BB','Barbados'],['BY','Belarus'],['BE','Belgium'],['BZ','Belize'],['BJ','Benin'],['BM','Bermuda'],['BT','Bhutan'],['BO','Bolivia'],['BQ','Bonaire, Sint Eustatius and Saba'],['BA','Bosnia and Herzegovina'],['BW','Botswana'],['BV','Bouvet Island'],['BR','Brazil'],['IO','British Indian Ocean Territory'],['BN','Brunei Darussalam'],['BG','Bulgaria'],['BF','Burkina Faso'],['BI','Burundi'],['CV','Cabo Verde'],['KH','Cambodia'],['CM','Cameroon'],['CA','Canada'],['KY','Cayman Islands'],['CF','Central African Republic'],['TD','Chad'],['CL','Chile'],['CN','China'],['CX','Christmas Island'],['CC','Cocos (Keeling) Islands'],['CO','Colombia'],['KM','Comoros'],['CG','Congo'],['CD','Congo (Democratic Republic)'],['CK','Cook Islands'],['CR','Costa Rica'],['CI','Côte d’Ivoire'],['HR','Croatia'],['CU','Cuba'],['CW','Curaçao'],['CY','Cyprus'],['CZ','Czechia'],['DK','Denmark'],['DJ','Djibouti'],['DM','Dominica'],['DO','Dominican Republic'],['EC','Ecuador'],['EG','Egypt'],['SV','El Salvador'],['GQ','Equatorial Guinea'],['ER','Eritrea'],['EE','Estonia'],['SZ','Eswatini'],['ET','Ethiopia'],['FK','Falkland Islands'],['FO','Faroe Islands'],['FJ','Fiji'],['FI','Finland'],['FR','France'],['GF','French Guiana'],['PF','French Polynesia'],['TF','French Southern Territories'],['GA','Gabon'],['GM','Gambia'],['GE','Georgia'],['DE','Germany'],['GH','Ghana'],['GI','Gibraltar'],['GR','Greece'],['GL','Greenland'],['GD','Grenada'],['GP','Guadeloupe'],['GU','Guam'],['GT','Guatemala'],['GG','Guernsey'],['GN','Guinea'],['GW','Guinea-Bissau'],['GY','Guyana'],['HT','Haiti'],['HM','Heard Island and McDonald Islands'],['VA','Holy See'],['HN','Honduras'],['HK','Hong Kong'],['HU','Hungary'],['IS','Iceland'],['IN','India'],['ID','Indonesia'],['IR','Iran'],['IQ','Iraq'],['IE','Ireland'],['IM','Isle of Man'],['IL','Israel'],['IT','Italy'],['JM','Jamaica'],['JP','Japan'],['JE','Jersey'],['JO','Jordan'],['KZ','Kazakhstan'],['KE','Kenya'],['KI','Kiribati'],['KP','Korea (North)'],['KR','Korea (South)'],['KW','Kuwait'],['KG','Kyrgyzstan'],['LA','Laos'],['LV','Latvia'],['LB','Lebanon'],['LS','Lesotho'],['LR','Liberia'],['LY','Libya'],['LI','Liechtenstein'],['LT','Lithuania'],['LU','Luxembourg'],['MO','Macao'],['MG','Madagascar'],['MW','Malawi'],['MY','Malaysia'],['MV','Maldives'],['ML','Mali'],['MT','Malta'],['MH','Marshall Islands'],['MQ','Martinique'],['MR','Mauritania'],['MU','Mauritius'],['YT','Mayotte'],['MX','Mexico'],['FM','Micronesia'],['MD','Moldova'],['MC','Monaco'],['MN','Mongolia'],['ME','Montenegro'],['MS','Montserrat'],['MA','Morocco'],['MZ','Mozambique'],['MM','Myanmar'],['NA','Namibia'],['NR','Nauru'],['NP','Nepal'],['NL','Netherlands'],['NC','New Caledonia'],['NZ','New Zealand'],['NI','Nicaragua'],['NE','Niger'],['NG','Nigeria'],['NU','Niue'],['NF','Norfolk Island'],['MK','North Macedonia'],['MP','Northern Mariana Islands'],['NO','Norway'],['OM','Oman'],['PK','Pakistan'],['PW','Palau'],['PS','Palestine'],['PA','Panama'],['PG','Papua New Guinea'],['PY','Paraguay'],['PE','Peru'],['PH','Philippines'],['PN','Pitcairn'],['PL','Poland'],['PT','Portugal'],['PR','Puerto Rico'],['QA','Qatar'],['RE','Réunion'],['RO','Romania'],['RU','Russia'],['RW','Rwanda'],['BL','Saint Barthélemy'],['SH','Saint Helena, Ascension and Tristan da Cunha'],['KN','Saint Kitts and Nevis'],['LC','Saint Lucia'],['MF','Saint Martin (French part)'],['PM','Saint Pierre and Miquelon'],['VC','Saint Vincent and the Grenadines'],['WS','Samoa'],['SM','San Marino'],['ST','Sao Tome and Principe'],['SA','Saudi Arabia'],['SN','Senegal'],['RS','Serbia'],['SC','Seychelles'],['SL','Sierra Leone'],['SG','Singapore'],['SX','Sint Maarten (Dutch part)'],['SK','Slovakia'],['SI','Slovenia'],['SB','Solomon Islands'],['SO','Somalia'],['ZA','South Africa'],['GS','South Georgia and the South Sandwich Islands'],['SS','South Sudan'],['ES','Spain'],['LK','Sri Lanka'],['SD','Sudan'],['SR','Suriname'],['SJ','Svalbard and Jan Mayen'],['SE','Sweden'],['CH','Switzerland'],['SY','Syria'],['TW','Taiwan'],['TJ','Tajikistan'],['TZ','Tanzania'],['TH','Thailand'],['TL','Timor-Leste'],['TG','Togo'],['TK','Tokelau'],['TO','Tonga'],['TT','Trinidad and Tobago'],['TN','Tunisia'],['TR','Türkiye'],['TM','Turkmenistan'],['TC','Turks and Caicos Islands'],['TV','Tuvalu'],['UG','Uganda'],['UA','Ukraine'],['AE','United Arab Emirates'],['GB','United Kingdom'],['US','United States'],['UM','United States Minor Outlying Islands'],['UY','Uruguay'],['UZ','Uzbekistan'],['VU','Vanuatu'],['VE','Venezuela'],['VN','Vietnam'],['VG','Virgin Islands (British)'],['VI','Virgin Islands (U.S.)'],['WF','Wallis and Futuna'],['EH','Western Sahara'],['YE','Yemen'],['ZM','Zambia'],['ZW','Zimbabwe']
  ];
  if (countrySel) COUNTRIES.forEach(function (c) {
    var o = document.createElement('option');
    o.value = c[0]; o.textContent = c[1] + ' (' + c[0] + ')';
    countrySel.appendChild(o);
  });

  var GT = window.GT;
  if (!GT || !GT.isConfigured) {
    hide(loading); show(signedout);
    signedout.querySelector('p').textContent = 'Profiles aren’t set up on this site yet — check back soon.';
    return;
  }

  // Base columns exist since Phase 1; the rest arrive with schema-profiles.sql.
  var BASE_COLS = 'id, username, display_name, avatar_url, created_at';
  var RICH_COLS = BASE_COLS + ', bio, website, callsign, country, region, city';
  var hasRich = true;
  var loadedFor = null;

  function fillForm(p) {
    field('display_name').value = p.display_name || '';
    field('username').value = p.username || '';
    avatarImg.src = p.avatar_url || DEFAULT_AVATAR;
    if (p.username) viewLink.setAttribute('href', '/u/?u=' + encodeURIComponent(p.username));
    if (p.created_at) {
      try {
        root.querySelector('[data-since-val]').textContent = new Date(p.created_at).toLocaleDateString(undefined, { year: 'numeric', month: 'long' });
        show(sinceBox);
      } catch (e) {}
    }
    if (hasRich) {
      field('callsign').value = p.callsign || '';
      if (countrySel) countrySel.value = p.country || '';
      field('region').value = p.region || '';
      field('city').value = p.city || '';
      field('bio').value = p.bio || '';
      field('website').value = p.website || '';
    }
  }

  // Fields that only make sense once the rich columns exist. If the Phase 4 SQL
  // isn't applied yet, hide them so /profile/ still saves name + username.
  function hideRichFields() {
    ['callsign', 'region', 'city', 'bio', 'website'].forEach(function (n) {
      var el = field(n); var lbl = el && el.closest('label'); if (lbl) hide(lbl);
    });
    var loc = root.querySelector('.profile-edit__location'); if (loc) hide(loc);
  }

  function loadProfile() {
    if (!GT.user || loadedFor === GT.user.id) return;
    loadedFor = GT.user.id;
    GT.supabase.from('profiles').select(RICH_COLS).eq('id', GT.user.id).single()
      .then(function (res) {
        if (res.error && /column|does not exist|schema cache/i.test(res.error.message || '')) {
          hasRich = false; hideRichFields();
          return GT.supabase.from('profiles').select(BASE_COLS).eq('id', GT.user.id).single();
        }
        return res;
      })
      .then(function (res) {
        if (res && res.data) fillForm(res.data);
      });
  }

  function render() {
    hide(loading);
    if (GT.user) { hide(signedout); show(form); loadProfile(); }
    else { hide(form); show(signedout); loadedFor = null; }
  }
  GT.onChange(render);
  if (GT.ready && GT.ready.then) GT.ready.then(render);

  // Avatar upload -> Storage bucket `avatars`, path `<uid>/avatar`.
  var MAX_AVATAR = 2 * 1024 * 1024;
  if (avatarIn) avatarIn.addEventListener('change', function () {
    var file = avatarIn.files && avatarIn.files[0];
    if (!file || !GT.user) return;
    if (!/^image\/(png|jpeg|webp)$/.test(file.type)) { setMsg(avatarMsg, 'Use a PNG, JPEG, or WebP image.', false); return; }
    if (file.size > MAX_AVATAR) { setMsg(avatarMsg, 'That image is over 2 MB — pick a smaller one.', false); return; }
    setMsg(avatarMsg, 'Uploading…', true);
    var path = GT.user.id + '/avatar';
    GT.supabase.storage.from('avatars').upload(path, file, { upsert: true, contentType: file.type })
      .then(function (res) {
        if (res.error) { setMsg(avatarMsg, 'Avatar uploads aren’t set up yet — the rest of your profile still saves.', false); return; }
        var pub = GT.supabase.storage.from('avatars').getPublicUrl(path);
        var url = (pub && pub.data && pub.data.publicUrl ? pub.data.publicUrl : '') + '?v=' + Date.now();
        avatarImg.src = url;
        return GT.supabase.from('profiles').upsert({ id: GT.user.id, avatar_url: url }).then(function (r2) {
          setMsg(avatarMsg, r2.error ? r2.error.message : 'Picture updated.', r2.error ? false : true);
        });
      });
  });

  // Validation mirrors the DB CHECKs in schema-profiles.sql.
  var RE_USER = /^[a-zA-Z0-9_-]{3,30}$/;
  var RE_CALL = /^[A-Z0-9/]{3,10}$/;
  var RE_SITE = /^https?:\/\/[^ ]{3,200}$/i;

  form.addEventListener('submit', function (ev) {
    ev.preventDefault();
    if (!GT.user) return;
    var dn = field('display_name').value.trim();
    var un = field('username').value.trim().toLowerCase();
    var row = { id: GT.user.id, display_name: dn || null, username: un || null };

    if (un && !RE_USER.test(un)) { setMsg(msg, 'Username must be 3–30 characters: letters, numbers, _ or -.', false); return; }

    if (hasRich) {
      var call = field('callsign').value.trim().toUpperCase();
      var site = field('website').value.trim();
      if (call && !RE_CALL.test(call)) { setMsg(msg, 'Call sign should be 3–10 characters (letters, numbers, /).', false); return; }
      if (site && !RE_SITE.test(site)) { setMsg(msg, 'Website should start with http:// or https://.', false); return; }
      row.callsign = call || null;
      row.country  = (countrySel && countrySel.value) || null;
      row.region   = field('region').value.trim() || null;
      row.city     = field('city').value.trim() || null;
      row.bio      = field('bio').value.trim() || null;
      row.website  = site || null;
      field('callsign').value = call;   // reflect the uppercasing
    }

    setMsg(msg, 'Saving…', true);
    GT.supabase.from('profiles').upsert(row).then(function (res) {
      if (res.error) {
        setMsg(msg, /duplicate|unique/i.test(res.error.message) ? 'That username is taken.' : res.error.message, false);
      } else {
        setMsg(msg, 'Saved.', true);
        if (un) viewLink.setAttribute('href', '/u/?u=' + encodeURIComponent(un));
      }
    });
  });
});
</script>
