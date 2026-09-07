<script setup lang="ts">
// ISO 3166-1 alpha-2 countries and territories; names are localized by the browser.
const countryCodes =
  "AD AE AF AG AI AL AM AO AQ AR AS AT AU AW AX AZ BA BB BD BE BF BG BH BI BJ BL BM BN BO BQ BR BS BT BV BW BY BZ CA CC CD CF CG CH CI CK CL CM CN CO CR CU CV CW CX CY CZ DE DJ DK DM DO DZ EC EE EG EH ER ES ET FI FJ FK FM FO FR GA GB GD GE GF GG GH GI GL GM GN GP GQ GR GS GT GU GW GY HK HM HN HR HT HU ID IE IL IM IN IO IQ IR IS IT JE JM JO JP KE KG KH KI KM KN KP KR KW KY KZ LA LB LC LI LK LR LS LT LU LV LY MA MC MD ME MF MG MH MK ML MM MN MO MP MQ MR MS MT MU MV MW MX MY MZ NA NC NE NF NG NI NL NO NP NR NU NZ OM PA PE PF PG PH PK PL PM PN PR PS PT PW PY QA RE RO RS RU RW SA SB SC SD SE SG SH SI SJ SK SL SM SN SO SR SS ST SV SX SY SZ TC TD TF TG TH TJ TK TL TM TN TO TR TT TV TW TZ UA UG UM US UY UZ VA VC VE VG VI VN VU WF WS YE YT ZA ZM ZW".split(
    " ",
  );
const regionNames = new Intl.DisplayNames(["zh-CN"], { type: "region" });
const countries = countryCodes
  .map((code) => ({ code, name: regionNames.of(code) ?? code }))
  .sort((a, b) => a.name.localeCompare(b.name, "zh-CN"));
const commonCountries = ["PH", "JP", "US", "BO"];
const model = defineModel<string>({ required: true });
withDefaults(
  defineProps<{ label?: string; id?: string; shortcuts?: boolean; disabled?: boolean }>(),
  { label: "国家或地区", shortcuts: false, disabled: false },
);
</script>

<template>
  <div class="country-select">
    <label class="text-field" :for="id"
      >{{ label
      }}<select :id="id" v-model="model" :disabled="disabled">
        <option value="" disabled>请选择国家或地区</option>
        <option v-for="country in countries" :key="country.code" :value="country.code">
          {{ country.name }}
        </option>
      </select></label
    >
    <div v-if="shortcuts" class="country-shortcuts">
      <span>常用</span
      ><button
        v-for="code in commonCountries"
        :key="code"
        type="button"
        :disabled="disabled"
        :aria-pressed="model === code"
        @click="model = code"
      >
        {{ regionNames.of(code) }}
      </button>
    </div>
  </div>
</template>

<style scoped src="./CountrySelect.css"></style>
