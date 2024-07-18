import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

/**
 * Store for tenantId. Compatible with localStorage.
 */
export const useTenantIdStore = defineStore('tenantId', () => {
    const tenantId = ref(localStorage.getItem('tenantId') || '');

    /**
     * Sets tenantId to new value and saves it to localStorage.
     *
     * @param newTenantId - New tenantId to set.
     */
    function setTenantId(newTenantId: string) {
        tenantId.value = newTenantId;
        localStorage.setItem('tenantId', newTenantId);
    }

    /**
     * Removes tenantId from store and localStorage.
     */
    function removeTenantId () {
        tenantId.value = '';
        localStorage.removeItem('tenantId');
    }

    return { tenantId, setTenantId, removeTenantId }
})
