<template>
  <div class="container">
    <div class="col-286 above-bottom-button">
      <p class="header">{{$t('mining.spendable_balance')}}:</p>
      <p class="spendableBalance">
        <a class="tiny" @click="refreshBalance">
          <svg
            width="16"
            height="16"
            version="1.1"
            id="Capa_1"
            xmlns="http://www.w3.org/2000/svg"
            xmlns:xlink="http://www.w3.org/1999/xlink"
            x="0px"
            y="0px"
            viewBox="0 0 458.186 458.186"
            style="enable-background:new 0 0 458.186 458.186;"
            xml:space="preserve"
          >
            <g>
              <g>
                <path
                  style="fill: #048652"
                  d="M445.651,201.95c-1.485-9.308-10.235-15.649-19.543-14.164c-9.308,1.485-15.649,10.235-14.164,19.543
										c0.016,0.102,0.033,0.203,0.051,0.304c17.38,102.311-51.47,199.339-153.781,216.719c-102.311,17.38-199.339-51.47-216.719-153.781
										S92.966,71.232,195.276,53.852c62.919-10.688,126.962,11.29,170.059,58.361l-75.605,25.19
										c-8.944,2.976-13.781,12.638-10.806,21.582c0.001,0.002,0.002,0.005,0.003,0.007c2.976,8.944,12.638,13.781,21.582,10.806
										c0.003-0.001,0.005-0.002,0.007-0.002l102.4-34.133c6.972-2.322,11.675-8.847,11.674-16.196v-102.4
										C414.59,7.641,406.949,0,397.523,0s-17.067,7.641-17.067,17.067v62.344C292.564-4.185,153.545-0.702,69.949,87.19
										s-80.114,226.911,7.779,310.508s226.911,80.114,310.508-7.779C435.905,339.799,457.179,270.152,445.651,201.95z"
                />
              </g>
            </g>
          </svg>
        </a>
        &nbsp;{{balance}} VTC
        <a class="tiny" @click="copyAddress" v-bind:title="$t('mining.copy_address')">
          <svg width="16" height="16" version="1.1" id="Capa_1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" x="0px" y="0px"
          viewBox="0 0 368.008 368.008" style="enable-background:new 0 0 368.008 368.008;" xml:space="preserve">
            <g>
              <g>
                <path style="fill: #048652" d="M368,88.004c0-1.032-0.224-2.04-0.6-2.976c-0.152-0.376-0.416-0.664-0.624-1.016c-0.272-0.456-0.472-0.952-0.832-1.352
                  l-72.008-80c-1.512-1.688-3.672-2.656-5.944-2.656h-15.648c-0.232,0-0.472,0-0.704,0H151.992c-13.232,0-24,10.768-24,24v40H24
                  c-13.232,0-24,10.768-24,24v256c0,13.232,10.768,24,24,24h192c13.232,0,24-10.768,24-24v-40h104c13.232,0,24-10.768,24-24v-175.96
                  c0-0.016,0.008-0.024,0.008-0.04L368,88.004z M224,344.004c0,4.408-3.592,8-8,8H24c-4.408,0-8-3.592-8-8v-256c0-4.408,3.592-8,8-8
                  h104v88c0,4.416,3.584,8,8,8h88V344.004z M224,160.004h-80v-80h4.688L224,155.324V160.004z M352,280.004c0,4.416-3.592,8-8,8H240
                  v-119.64c0-0.12,0.008-0.24,0.008-0.36l-0.008-16c0,0,0-0.008,0-0.024c-0.008-2.12-0.832-4.04-2.184-5.464
                  c0-0.016-0.024-0.016-0.016-0.016c0,0-0.008-0.008-0.008-0.016c-0.008,0-0.016-0.008-0.016-0.016
                  c-0.032-0.032-0.072-0.072-0.112-0.112l-80-80c-1.504-1.504-3.544-2.352-5.664-2.352h-8.008v-40c0-4.408,3.592-8,8-8h112v88
                  c0,4.416,3.584,8,8,8H352V280.004z M352,96.004h-72.008v-80h4.44L352,91.076V96.004z"/>
              </g>
            </g>
          </svg>
        </a>
      </p>
      <p class="immatureBalance" v-if="balanceImmature != '0.00000000'">
        (
        <span style="opacity: 1">{{balanceImmature}} VTC</span>
        {{$t('mining.still_maturing')}})
      </p>
      <p class="poolBalance" v-if="balancePendingPool != '0.00000000'">
        (
        <span style="opacity: 1">{{balancePendingPool}} VTC</span>
        {{$t('mining.pending_pool_payout')}})
      </p>
<!--      <p class="poolBalance" v-if="balancePendingPool == '0.00000000'">-->
<!--        {{$t('mining.pending_payout_info_unavailable')}}-->
<!--      </p>-->
      <div class="notices" v-if="showRpcNotice || showClockNotice">
      <div class="clockNotice" v-if="showClockNotice">
        <a class="dismiss" @click="clockNoticeDismissed = true" v-bind:title="$t('generic.close')">&times;</a>
        <i18n path="mining.clock_notice" tag="span">
          <template v-slot:title><b>{{ $t(clockNotice.ahead ? 'mining.clock_notice_fast' : 'mining.clock_notice_slow', {minutes: clockNotice.minutes}) }}</b></template>
        </i18n>
      </div>
      <div class="rpcNotice" v-if="showRpcNotice">
        <a class="dismiss" @click="rpcNoticeDismissed = true" v-bind:title="$t('generic.close')">&times;</a>
        <i18n path="mining.rpc_notice" tag="span">
          <template v-slot:title><b>{{ $t('mining.rpc_notice_found') }}</b></template>
          <template v-slot:setting><code>{{ rpcNotice.sample }}</code></template>
          <template v-slot:file><span class="confFile" v-bind:title="rpcNotice.confPath">vertcoin.conf</span></template>
        </i18n>
      </div>
      </div>
      <p class="pool">
        <span style="opacity: 1">{{$t('mining.active_pool')}}: {{activePool}} <span v-if="poolFee != '0.0%'">({{$t('mining.pool_fee')}}: {{poolFee}})</span></span>
        <span class="coreNode" v-if="nodeMode.mode === 'full' && nodeMode.coreResponding">&#10003; {{ $t('mining.core_node_in_use') }}</span>
        <span class="coreNode down" v-if="nodeMode.mode === 'full' && !nodeMode.coreResponding">{{ $t('mining.core_node_not_responding') }}</span>
        <br><a class="link" @click="payoutInformation">{{ $t('mining.payout_information') }}</a>
      </p>
      
      <p class="spacer">&nbsp;</p>
      <p v-if="runningMiners === 0">{{$t('mining.waiting_for_miners')}}</p>
      <p v-if="runningMiners > 0" class="header">{{$t('mining.expected_earnings_24h')}}:</p>
      <p
        v-if="runningMiners > 0 && !estimating"
        class="earning"
      >~{{avgearn}} ({{hashrate}})</p>
      <p
        v-if="runningMiners > 0 && estimating"
        class="earning"
      >{{$t('mining.estimating')}}{{spinner}}</p>
      <p class="bottom-button">
        <a class="button" v-if="stopping">{{spinner}}</a>
        <a class="button" @click="stop" v-if="!stopping">{{$t('mining.stop_mining')}}</a>
      </p>
    </div>
  </div>
</template>

<script>
export default {
  props: {
    active: {type: Boolean, default: false},
  },
  watch: {
    active: function(now) {
      if (now) {
        this.refreshRpcNotice();
      }
    },
  },
  data() {
    return {
      hashrate: "0.00 MH/s",
      avgearn: "0.00 VTC",
      netHash: "",
      gpu: "",
      testnet: false,
      wallet: "",
      balance: "0.00000000",
      balanceImmature: "0.00000000",
      balancePendingPool: "0.00000000",
      runningMiners: 0,
      spinner: "...",
      stopping: false,
      poolFee: "0.0%",
      activePool: "",
      address:"",
      rpcNotice: {show: false, confPath: "", sample: ""},
      rpcNoticeDismissed: false,
      clockNotice: {show: false, minutes: 0, ahead: false},
      clockNoticeDismissed: false,
      rpcNoticeTimer: null,
      nodeMode: {mode: "", coreResponding: false},
    };
  },
  beforeDestroy() {
    if (this.rpcNoticeTimer !== null) {
      window.clearInterval(this.rpcNoticeTimer);
      this.rpcNoticeTimer = null;
    }
  },
  mounted() {
    var self = this;
    window.setInterval(() => {
      var newSpinner = self.spinner + ".";
      if (newSpinner.length > 5) {
        newSpinner = ".";
      }
      self.spinner = newSpinner;
    }, 1000);
    this.refreshRpcNotice();
    // The notice only shows while Vertcoin Core runs, so check again now and then.
    this.rpcNoticeTimer = window.setInterval(() => {
      if (self.active) {
        self.refreshRpcNotice();
      }
    }, 30000);
    window.backend.Backend.GetTestnet().then(result => {
      self.testnet = result;
    });
    window.setInterval(() => {
      window.backend.Backend.GetPoolName().then(result => {
        self.activePool = result;
      });
      window.backend.Backend.GetPoolFee().then(result => {
        self.poolFee = result;
      });
      window.backend.Backend.Address().then(result => {
        self.address = result;
      });
    }, 5000);
    window.backend.Backend.GetPoolName().then(result => {
      self.activePool = result;
    });
    window.backend.Backend.GetPoolFee().then(result => {
      self.poolFee = result;
    });
    window.wails.Events.On("hashRate", result => {
      self.hashrate = result;
    });
    window.wails.Events.On("runningMiners", result => {
      self.runningMiners = result;
    });
    window.wails.Events.On("networkHashRate", result => {
      self.netHash = result;
    });
    window.wails.Events.On("avgEarnings", result => {
      self.avgearn = result;
    });
    window.wails.Events.On("balance", result => {
      self.balance = result;
    });
    window.wails.Events.On("balanceImmature", result => {
      self.balanceImmature = result;
    });
    window.wails.Events.On("balancePendingPool", result => {
      self.balancePendingPool = result;
    });
    window.backend.Backend.RefreshBalance();
    window.backend.Backend.RefreshHashrate();
    window.backend.Backend.RefreshRunningState();
  },
  computed: {
    showRpcNotice() {
      return this.rpcNotice.show && !this.rpcNoticeDismissed;
    },
    showClockNotice() {
      return this.clockNotice.show && !this.clockNoticeDismissed;
    },
    // The backend reports no hashrate as "0.00 kH/s", so check the number, not the text.
    estimating() {
      return !(parseFloat(this.hashrate) > 0);
    },
  },
  methods: {
    // Refreshes what the screen says about Vertcoin Core: the RPC notice and
    // whether the node is mining through it.
    refreshRpcNotice: function() {
      var self = this;
      window.backend.Backend.GetRPCNotice().then(result => {
        self.rpcNotice = result;
      });
      window.backend.Backend.GetNodeMode().then(result => {
        self.nodeMode = result;
      });
      // The backend checks the clock at most every 10 minutes.
      window.backend.Backend.GetClockNotice().then(result => {
        self.clockNotice = result;
      });
    },
    stop: function() {
      var self = this;
      this.stopping = true;
      window.backend.Backend.StopMining().then(() => {
        self.stopping = false;
        self.$emit("stop-mining");
      });
    },
    refreshBalance: function() {
      window.backend.Backend.RefreshBalance();
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
    sendMoney: function() {
      this.$emit("send");
    },
    payoutInformation: function() {
      window.backend.Backend.PayoutInformation();
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
/* One line in the gap under the tab bar, out of the centered column's flow,
   so it never pushes Stop Mining into the footer at 800x400. */
div.notices {
  position: fixed;
  top: 62px;
  left: 50%;
  transform: translateX(-50%);
  -ms-transform: translateX(-50%);
}
div.rpcNotice,
div.clockNotice {
  position: relative;
  margin-bottom: 2px;
  box-sizing: border-box;
  white-space: nowrap;
  border: 1px solid #d0a000;
  color: #d0a000;
  padding: 2px 20px 2px 8px;
  text-align: left;
  line-height: 14px;
  font-size: 11px;
}
div.rpcNotice code {
  font-family: "Courier New", Courier, monospace;
  user-select: text;
}
div.rpcNotice span.confFile {
  text-decoration: underline dotted;
  cursor: help;
}
div.rpcNotice a.dismiss,
div.clockNotice a.dismiss {
  position: absolute;
  top: 1px;
  right: 5px;
  font-size: 14px;
  line-height: 14px;
  cursor: pointer;
  opacity: 0.8;
}
div.rpcNotice a.dismiss:hover,
div.clockNotice a.dismiss:hover {
  opacity: 1;
}
a.tiny:hover {
  opacity: 1;
  transition: 500ms;
}
a.tiny {
  opacity: 0.6;
  background: transparent;
  display: inline;
  z-index: 500;
  margin: 0px;
  cursor: pointer;
  border: 0px;
}

p.spendableBalance,
p.earning {
  margin: 0;
  padding: 0;
  font-size: 20px;
}
p.immatureBalance,
p.poolBalance,
p.netHash {
  margin: 0;
  padding: 0;
  font-size: 12px;
  opacity: 0.6;
}
p.spacer {
  padding: 0px;
  margin: 5px;
}
span.coreNode {
  color: #048652;
  white-space: nowrap;
  margin-left: 4px;
}
span.coreNode.down {
  color: #d0a000;
}
p.header {
  margin-bottom: 0;
  padding-bottom: 5px;
  opacity: 0.6;
}
p.fork {
  display: block;
  border: 2px solid #d0a000;
  color: #d0a000;
  font-weight: bold;
}

p.fork>a, p.fork>a:active, p.fork>a:visited {
  color: #d0a000;
  font-weight: bold;
  padding: 5px;
  cursor: pointer;
}

a.link {
  font-size: 14px;
  opacity: 0.6;
  text-decoration: underline;
  cursor: pointer;
}

</style>
