<script lang="ts">
    import { tick } from 'svelte';

    interface Props {
        // src is the image as a data URL produced by LoadDocument.
        src: string;
        // name is the file name, used for the alt text and errors.
        name: string;
        // onSize reports the image's natural pixel size, which the status bar
        // prints beside the language.
        onSize?: (width: number, height: number) => void;
    }

    let { src, name, onSize }: Props = $props();

    // Fit is exact: the image is never blown up beyond its natural size, so
    // small icons stay sharp. The padding keeps a fitted image off the pane
    // edges.
    const PADDING = 16;
    const MIN_SCALE = 0.01;
    const MAX_SCALE = 16;
    // A wheel notch is roughly 100 deltaY, which works out to a ~22% step.
    const ZOOM_SPEED = 0.002;

    let viewport = $state<HTMLDivElement>();
    let image = $state<HTMLImageElement>();
    let paneWidth = $state(0);
    let paneHeight = $state(0);
    let naturalWidth = $state(0);
    let naturalHeight = $state(0);
    let fit = $state(true);
    let zoom = $state(1);
    let failed = $state(false);

    let availableWidth = $derived(Math.max(1, paneWidth - 2 * PADDING));
    let availableHeight = $derived(Math.max(1, paneHeight - 2 * PADDING));
    let fitScale = $derived(
        naturalWidth === 0
            ? 1
            : Math.min(1, availableWidth / naturalWidth, availableHeight / naturalHeight),
    );
    let scale = $derived(fit ? fitScale : zoom);
    let displayWidth = $derived(naturalWidth * scale);
    let displayHeight = $derived(naturalHeight * scale);

    function clampScale(value: number): number {
        return Math.min(MAX_SCALE, Math.max(MIN_SCALE, value));
    }

    // change keeps the image pixel under (clientX, clientY) fixed while the
    // image resizes, so zooming and switching to actual size do not jump.
    async function keepPointUnderCursor(
        clientX: number,
        clientY: number,
        change: () => void,
    ): Promise<void> {
        const element = image;
        if (element === undefined || naturalWidth === 0 || naturalHeight === 0) {
            return;
        }
        const before = element.getBoundingClientRect();
        if (before.width === 0 || before.height === 0) {
            return;
        }
        // Clamp so a double-click on the pane background anchors to the nearest
        // image edge instead of scrolling past it.
        const pointX = Math.min(1, Math.max(0, (clientX - before.left) / before.width));
        const pointY = Math.min(1, Math.max(0, (clientY - before.top) / before.height));

        change();
        await tick();

        const after = element.getBoundingClientRect();
        const target = viewport;
        if (target === undefined || after.width === 0 || after.height === 0) {
            return;
        }
        target.scrollLeft += after.left + pointX * after.width - clientX;
        target.scrollTop += after.top + pointY * after.height - clientY;
    }

    function handleWheel(event: WheelEvent): void {
        // Cmd zooms on macOS; Ctrl also works for trackpad pinch, which
        // reports Ctrl. A plain wheel is left to the scroll container so tall
        // images can be panned without a modifier.
        if (!event.metaKey && !event.ctrlKey) {
            return;
        }
        if (naturalWidth === 0 || naturalHeight === 0) {
            return;
        }

        event.preventDefault();
        const next = clampScale(
            (fit ? fitScale : zoom) * Math.exp(-event.deltaY * ZOOM_SPEED),
        );
        void keepPointUnderCursor(event.clientX, event.clientY, () => {
            zoom = next;
            fit = false;
        });
    }

    function toggleSize(event: MouseEvent): void {
        if (naturalWidth === 0 || naturalHeight === 0) {
            return;
        }
        void keepPointUnderCursor(event.clientX, event.clientY, () => {
            if (fit) {
                fit = false;
                zoom = 1;
            } else {
                fit = true;
            }
        });
    }

    function handleLoad(): void {
        const element = image;
        if (element === undefined) {
            return;
        }
        naturalWidth = element.naturalWidth;
        naturalHeight = element.naturalHeight;
        fit = true;
        zoom = 1;
        failed = false;
        onSize?.(naturalWidth, naturalHeight);
    }

    function handleError(): void {
        failed = true;
    }
</script>

<div
    class="viewer"
    class:viewer--failed={failed}
    bind:this={viewport}
    bind:clientWidth={paneWidth}
    bind:clientHeight={paneHeight}
    onwheel={handleWheel}
    ondblclick={toggleSize}
    role="group"
    aria-label={`Image ${name}`}
>
    {#if failed}
        <p class="viewer__error">Cannot preview {name}.</p>
    {:else}
        <div class="viewer__stage">
            <img
                bind:this={image}
                class="viewer__image"
                class:viewer__image--ready={naturalWidth > 0}
                {src}
                alt={name}
                draggable="false"
                onload={handleLoad}
                onerror={handleError}
                style:width={`${displayWidth}px`}
                style:height={`${displayHeight}px`}
            />
        </div>
    {/if}
</div>

<style>
    .viewer {
        position: absolute;
        inset: 0;
        overflow: auto;
        background: var(--bg);
    }

    /* inline-flex shrinks to the image when it is larger than the pane, so the
       scroll container can reach every edge, and stretches to the pane when the
       image is smaller, keeping it centred. */
    .viewer__stage {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        min-width: 100%;
        min-height: 100%;
        padding: 16px;
    }

    .viewer__image {
        display: block;
        visibility: hidden;
        user-select: none;
        -webkit-user-drag: none;
    }

    .viewer__image--ready {
        visibility: visible;
    }

    .viewer__error {
        display: flex;
        align-items: center;
        justify-content: center;
        height: 100%;
        margin: 0;
        color: var(--fg-muted);
    }
</style>
