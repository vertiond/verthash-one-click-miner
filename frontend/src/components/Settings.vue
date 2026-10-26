<template>
  <div class="settings-container">
    <div class="col-settings">
      <div class="col-settings-sub">
        <p style="text-align: left" >
          <input type="checkbox" v-model="debugging" />
          {{ $t("settings.enable_debug") }}
          <br />
          <span class="subtext">{{ $t("settings.enable_debug_sub") }}</span>
        </p>
        <p style="text-align: left">
          <input type="checkbox" v-model="autoStart" />
          {{ $t("settings.auto_start") }}
          <br />
          <span class="subtext">{{ $t("settings.auto_start_sub") }}</span>
        </p>
      </div>
      <div class="col-settings-sub">
        <p style="text-align: left">
          <input type="checkbox" v-model="enableIntegrated" />
          {{ $t("settings.enable_integrated") }}
          <br />
          <span class="subtext">{{ $t("settings.enable_integrated_sub") }}</span>
        </p>
        <p style="text-align: left">
          {{ $t("settings.pool") }}:
          <br />
          <select style="width: 100%" name="pool" v-model="poolID">
            <option v-for="option in pools" v-bind:value="option.id" v-bind:key="option.id">
                {{ option.name }}
            </option>
          </select>
        </p>
      </div>
    </div>
    <div class="col-286 height-100">
      <p>
        <a class="button" @click="save">{{ $t("settings.save_n_restart") }}</a>
      </p>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      debugging: false,
      autoStart: false,
      enableIntegrated: true,
      poolID: -1,
      pools: [],
    };
  },
  created() {
    var self = this;
    window.backend.Backend.GetAutoStart().then(result => {
      self.autoStart = result;
      window.backend.Backend.GetDebugging().then(result => {
        self.debugging = result;
        window.backend.Backend.GetPools().then(result => {
          self.pools = result;
          window.backend.Backend.GetPool().then(result => {
            self.poolID = result;
            window.backend.Backend.GetEnableIntegrated().then(result => {
              self.enableIntegrated = result;
            });
          });
        });
      });
    });
  },
  methods: {
    save: function() {
      var self = this;
      window.backend.Backend.SetDebugging(self.debugging).then(() => {
        window.backend.Backend.SetAutoStart(self.autoStart).then(() => {
          window.backend.Backend.SetPool(self.poolID).then(() => {
            window.backend.Backend.SetEnableIntegrated(self.enableIntegrated).then( () => {
              self.$emit("committed");
            });
          });
        });
      });
    }
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
span.subtext {
  opacity: 0.6;
  font-size: 8pt;
}
</style>
