<script setup lang="ts">
import { onMounted } from 'vue';
/* @ts-ignore */
import Calendar from '@event-calendar/core';
/* @ts-ignore */
import TimeGrid from '@event-calendar/time-grid';
/* @ts-ignore */
import Interaction from '@event-calendar/interaction';

onMounted(() => {
    let ec = new Calendar({
        target: document.getElementById('calendar'),
        props: {
            plugins: [TimeGrid, Interaction],
            options: {
                view: 'timeGridWeek',
                events: [
                {
                    title: 'Möte',
                    start: '2024-07-19T10:00:00',
                    end: '2024-07-19T11:00:00',
                }
                ],
                buttonText: {
                today: 'Idag'
                },
                allDaySlot: false,
                firstDay: 1,
                locale: 'sv',
                nowIndicator: true,
                editable: true,
                selectable: true,
            }
        }
    });
})
</script>

<template>
    <div id="calendar"></div>
</template>

<style lang="scss" scoped>
@import "@/assets/styles/variables.scss";

#calendar {
  height: $viewport-height;
  overflow: auto;
  padding: 1rem;
}
</style>
