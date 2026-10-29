<template>
  <div class="container">
    <div v-if="checkStatus !== 'Failed'" class="col-286">
      <!-- Only the headline is centred; the lines under it hang below, so the
           headline stays put when they appear or disappear -->
      <div class="pinned">
      <p>{{checkStatus === null ? $t("checks.checking_mining_software") : (checkStatus === 'Failed' ? 'Failed' : $t("checks." + checkStatus)) }}</p>
      <div class="below">
      <div class="verthashProgress" v-if="checkStatus === 'verthash' && verthashProgress !== 0">
        <div class="progressBar">
          <div class="progress" v-bind:style="{width: verthashProgress + '%'}">&nbsp;</div>
        </div>
        <div class="progressText">{{Math.floor(verthashProgress)}}%</div>
      </div>
      <p class="elapsed" v-if="waitTimer !== null">{{ $tc("checks.node_peers", nodePeers, {count: nodePeers}) }} &middot; {{ elapsedText }}</p>
      <div class="nodeSlow" v-if="nodeSlow">
        <p>{{ $t("checks.node_slow") }}</p>
        <a class="button" @click="retryNode">{{ $t('generic.retry') }}</a>
      </div>
      </div>
      </div>
    </div>
    <div v-if="checkStatus === 'Failed'" class="col-wide">
      <!-- One line: white-space: pre-line would show template line breaks. -->
      <div class="failureReason" v-if="checkStatus === 'Failed'">{{ $t("checks.checks_failed") }}:<br />&nbsp;<br />{{failureReason}}</div>
      <p v-if="checkStatus === 'Failed'">
        <a class="button" @click="check">{{ $t('generic.retry') }}</a>
      </p>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      checkStatus: null,
      failureReason: "",
      verthashProgress: 0,
      nodePeers: 0,
      nodeSlow: false,
      elapsed: 0,
      waitTimer: null,
    };
  },
  watch: {
    // Count the time spent waiting for the node from its first state on, so
    // the wait is visible from the start rather than only once it is slow.
    checkStatus: function(status) {
      var waiting = typeof status === "string" && status.indexOf("node_") === 0 && status !== "node_ready";
      if (waiting) {
        this.startWait();
      } else {
        this.stopWait();
      }
    }
  },
  computed: {
    elapsedText: function() {
      var m = Math.floor(this.elapsed / 60);
      var s = this.elapsed % 60;
      return (m < 10 ? "0" : "") + m + ":" + (s < 10 ? "0" : "") + s;
    }
  },
  beforeDestroy() {
    this.stopWait();
  },
  mounted() {
    this.check();
    var self = this;
    window.wails.Events.On("checkStatus", result => {
      self.checkStatus = result;
    });
    window.wails.Events.On("nodePeers", result => {
      self.nodePeers = result;
    });
    window.wails.Events.On("verthashProgress", result => {
      self.verthashProgress = result;
    });
    window.wails.Events.On("nodeSlow", result => {
      self.nodeSlow = result === true || result === "true";
      if (!self.nodeSlow) {
        // A retry restarts the node, so the wait starts over too.
        self.elapsed = 0;
      }
    });
  },
  methods: {
    startWait: function() {
      var self = this;
      if (this.waitTimer !== null) {
        return;
      }
      this.elapsed = 0;
      this.waitTimer = window.setInterval(() => {
        self.elapsed++;
      }, 1000);
    },
    stopWait: function() {
      if (this.waitTimer !== null) {
        window.clearInterval(this.waitTimer);
        this.waitTimer = null;
      }
      this.nodeSlow = false;
      this.elapsed = 0;
    },
    retryNode: function() {
      window.backend.Backend.RetryNode();
    },
    check: function() {
      var self = this;

      window.backend.Backend.PerformChecks().then(result => {
        self.stopWait();
        if (result === "ok") {
          self.startMining();
        } else {
          self.$emit("checksFailed");
          self.failureReason = result;
        }
      });
    },
    startMining: function() {
      var self = this;
      window.backend.Backend.StartMining().then(() => {
        self.$emit("mining");
      });
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
div.failureReason {
  white-space: pre-line;
  height: 200px;
  overflow-y: auto;
  font-family: "Courier New", Courier, monospace;
  color: red;
  border: 1px solid red;
  width: 600px;
  margin: 0 auto;
}

div.pinned {
  position: relative;
}

div.below {
  position: absolute;
  top: 100%;
  left: 0;
  width: 100%;
}

p.elapsed {
  opacity: 0.6;
  font-size: 10pt;
}

div.verthashProgress {
  margin: 0 auto;
  width: 200px;
}

div.progressBar {
  border: 1px solid #048652;
  height: 10px;
  width: 100%;
  margin: 0px;
  padding: 0px;
  margin-bottom: 10px;
}

div.progress {
  float:left;
  background-color: #048652;
  margin: 0px;
  padding: 0px;
  height: 10px; 
}
</style>

