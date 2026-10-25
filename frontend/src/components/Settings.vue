<template>
  <div class="settings-container">
    <div class="col-settings" v-if="!showWarning">
      <div class="col-settings-sub" v-bind:style="{ visibility: loaded ? 'visible' : 'hidden' }">
        <p style="text-align: left">
          <input type="checkbox" v-model="enableIntegrated" />
          {{ $t("settings.enable_integrated") }}
          <br />
          <span class="subtext">{{ $t("settings.enable_integrated_sub") }}</span>
        </p>
        <!-- Label and list on one line, so the column fits in every language -->
        <p class="settings-row">
          {{ $t("settings.language") }}:
          <select name="language" v-model="language" @change="changeLanguage">
            <!-- In the system's language, so the way back is readable from any other -->
            <option value="">{{ $t("settings.language_system", systemLocale) }}</option>
            <option v-for="option in languages" v-bind:value="option.code" v-bind:key="option.code">
                {{ option.name }}
            </option>
          </select>
        </p>
        <p v-if="!poolDogeOnly" style="text-align: left">
          {{ $t("settings.payout") }}:
          <br />
          <select style="width: 100%" name="payout" v-model="payoutID" v-on:change="changePayoutOptions">
            <option v-for="option in payouts" v-bind:value="option.id" v-bind:key="option.id">
                {{ option.name }}
            </option>
          </select>
        </p>
        <!-- TODO: Improve address validation -->
        <p v-if="!poolDogeOnly && payoutID != 4">
          <input type="text" style="width:90%" class="critical-input"
            v-model="customAddress"
            v-bind:placeholder="$t('settings.customAddress')"
            v-on:change="sanitizeAddress"
            v-on:keyup.enter="saveFromAddress"
          />
        </p>
        <p v-if="poolDogeOnly" style="text-align: left">
          {{ $t("settings.doge_only") }}
          <br><a class="link" @click="copyAndActivate">{{ $t('settings.address_activation') }}</a>
          {{ $t("settings.mine_verthash_paid_doge") }}
        </p>
      </div>
      <div class="col-settings-sub" v-bind:style="{ visibility: loaded ? 'visible' : 'hidden' }">
<!--        <p style="text-align: left">-->
<!--          <input type="checkbox" v-model="testnet" />-->
<!--          {{ $t("settings.testnet") }}-->
<!--          <br />-->
<!--          <span class="subtext">{{ $t("settings.testnet_sub") }}</span>-->
<!--        </p>-->
        <p style="text-align: left">
          <input type="checkbox" v-model="autoStart" />
          {{ $t("settings.auto_start") }}
          <br />
          <span class="subtext">{{ $t("settings.auto_start_sub") }}</span>
        </p>
        <p style="text-align: left" >
          <input type="checkbox" v-model="debugging" />
          {{ $t("settings.enable_debug") }}
          <br />
          <span class="subtext">{{ $t("settings.enable_debug_sub") }}</span>
        </p>
        <p style="text-align: left">
          {{ $t("settings.pool") }}:
          <br />
          <select style="width: 100%" name="pool" v-model="poolID" v-on:change="updatePoolPayouts">
            <option v-for="option in pools" v-bind:value="option.id" v-bind:key="option.id">
                {{ option.name }}
            </option>
          </select>
        </p>
      </div>
    </div>
    <div class="col-300 height-100" v-if="!showWarning">
      <p class="bottom-button">
        <a class="button" @click="save">{{ $t("settings.save_n_restart") }}</a>
      </p>
    </div>
    <div class="col-286" v-if="showWarning">
      <div class="warning" v-if="closedSourceMiner && showWarning">
        <p>{{ $t("settings.closed_source_warning") }}</p>
      </div>
       <p>
        <a class="button" @click="toggleWarning">{{ $t("generic.close") }}</a>
      </p>
    </div>
  </div>
</template>

<script>
import { loadSettings, cachedSettings, forgetSettings } from "../settingsLoader.js";
export default {
  data() {
    return {
      closedSourceMiner: false,
      debugging: false,
      autoStart: false,
      enableIntegrated: true,
      showWarning: false,
      testnet: false,
      poolID: -1,
      payoutID: -1,
      network: "",
      pools: [],
      payouts: [],
      networks: [],
      renderPayouts: true,
      customAddress: "",
      address:"",
      // Address typed for each payout, by payout ID, so changing the payout doesn't lose the others
      savedAddresses: {},
      lastPayoutID: -1,
      // False until the first values arrive, so half-loaded ones (empty pool list,
      // payout address box) are never shown
      loaded: false,
      language: "",
      languages: [],
      systemLocale: "en",
    };
  },
  computed: {
    // The selected pool only pays out to the built-in Dogecoin wallet
    poolDogeOnly: function() {
      var self = this;
      return self.pools.some(function(p) { return p.id == self.poolID && p.dogeOnly; });
    }
  },
  created() {
    var self = this;
    var apply = function(settings) {
      for (var key in settings) {
        self[key] = settings[key];
      }
      // Work on a copy so edits that are never saved don't leak into the cached settings
      self.savedAddresses = Object.assign({}, settings.savedAddresses);
      if (self.customAddress) {
        self.savedAddresses[self.payoutID] = self.customAddress;
      }
      self.lastPayoutID = self.payoutID;
      self.loaded = true;
    };

    // Show the previous values right away and refresh them in the background
    var cached = cachedSettings();
    if (cached) {
      apply(cached);
    }
    loadSettings()
      .then(apply)
      .catch(function() {
        self.loaded = true;
      });
  },
  methods: {
    toggleWarning: function() {
      this.showWarning = !this.showWarning;
      var self = this;
      setTimeout(() => { self.showWarning = false; }, 5000);
    },
    updatePoolPayouts: function() {
      var self = this;
      window.backend.Backend.GetPayouts(self.poolID).then(result => { self.payouts = result; });
      window.backend.Backend.GetKickbackPayoutID(self.poolID, self.payoutID).then(result => {
        self.payoutID = result;
        self.switchPayoutAddress();
      });
    },
    // Remember the address for the payout we're leaving and bring back the one for the new payout
    switchPayoutAddress: function() {
      var self = this;
      if (self.payoutID == self.lastPayoutID) {
        return;
      }
      self.savedAddresses[self.lastPayoutID] = self.customAddress;
      self.customAddress = self.savedAddresses[self.payoutID] || "";
      self.lastPayoutID = self.payoutID;
    },
    sanitizeAddress: function() {
      var self = this;
      self.customAddress = self.customAddress.trim();
    },
    saveFromAddress: function() {
      this.sanitizeAddress();
      this.save();
    },
    changePayoutOptions: function() {
      var self = this;
      self.switchPayoutAddress();
      window.backend.Backend.GetNetworks(self.poolID, self.payoutID).then(result => { self.networks = result; });
    },
    copyAndActivate: function() {
      this.copyAddress();
      window.backend.Backend.ActivationInformation(this.poolID);
    },
    copyAddress: function() {
      var textArea = document.createElement("textarea");
      textArea.value = this.address;
      // textArea.style.display = "none";
      // Avoid scrolling to bottom
      textArea.style.top = "0";
      textArea.style.left = "0";
      textArea.style.position = "fixed";
    
      document.body.appendChild(textArea);
      textArea.focus();
      textArea.select();
    
      try {
        document.execCommand('copy');
      } catch(e) {
        // ignore
      }
    
      document.body.removeChild(textArea);
    },
    // The language applies and is saved as soon as it is picked, without Save & Restart
    changeLanguage: function() {
      var self = this;
      window.backend.Backend.SetLanguage(this.language).then(() => {
        window.backend.Backend.GetLocale().then(locale => {
          self.$i18n.locale = locale;
          forgetSettings();
        });
      });
    },
    save: function() {
      var self = this;
      self.savedAddresses[self.payoutID] = self.customAddress;
      window.backend.Backend.SetPayoutAddresses(JSON.stringify(self.savedAddresses));
      window.backend.Backend.SetClosedSource(this.closedSourceMiner).then(() => {
          window.backend.Backend.SetDebugging(self.debugging).then(() => {
            window.backend.Backend.SetAutoStart(self.autoStart).then(() => {
              window.backend.Backend.SetTestnet(self.testnet).then(() => {
                window.backend.Backend.SetEnableIntegrated(self.enableIntegrated).then( () => {
                  window.backend.Backend.SetPool(self.poolID).then(() => {
                    window.backend.Backend.SetCustomAddress(self.customAddress).then(() => {
                      window.backend.Backend.SetPayout(self.payoutID).then(() => {
                        window.backend.Backend.SetNetwork(self.network).then(() => {
                          forgetSettings();
                          self.$emit("committed");
                        });
                      });
                    });
                  });
                });
              });
            });
          });
        }
      );
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
div.warning {
  border: 2px solid #d0a000;
  color: #d0a000;
  width: 100%;
  padding: 5px 10px;
  text-align: justify;
  line-height: 10pt;
  font-size: 10pt;
}
a.warning {
  display: block; 
  float:right;
  color: #d0a000;
  cursor: pointer;
  text-decoration: underline;
}
div.warning p {
  margin: 0px;
  padding: 0px;
}
span.subtext {
  opacity: 0.6;
  font-size: 8pt;
}
.critical-input::placeholder {
  color: red;
}
a.link {
  font-size: 14px;
  opacity: 1;
  text-decoration: underline;
  cursor: pointer;
  text-align: left;
}
</style>
