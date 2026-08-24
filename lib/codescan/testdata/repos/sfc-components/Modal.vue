<template>
  <!-- TODO(baseline/dialog): replace modal container with native dialog -->
  <div class="modal-dialog">
    <slot></slot>
  </div>
</template>

<script setup lang="ts">
// TODO(baseline/popover, newly): use popover for action menu
import { ref } from 'vue';
</script>

<style scoped>
/* TODO(baseline/anchor-positioning): position modal anchored to trigger */
.modal-dialog {
  position: fixed;
}
</style>
