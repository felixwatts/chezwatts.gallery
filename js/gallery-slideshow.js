/* global GallerySlideshow */
var GallerySlideshow = (function () {
    'use strict';

    function preload(url) {
        return new Promise(function (resolve, reject) {
            var img = new Image();
            img.onload = function () { resolve(img); };
            img.onerror = reject;
            img.src = url;
        });
    }

    function init(containerId, jsonScriptId, options) {
        options = options || {};
        var interval = options.interval !== undefined ? options.interval : 7000;
        var fadeMs = options.fadeMs !== undefined ? options.fadeMs : 700;
        var pauseOnHover = options.pauseOnHover !== false;
        var arrowKeys = options.arrowKeys !== false;

        var container = document.getElementById(containerId);
        var scriptEl = document.getElementById(jsonScriptId);
        if (!container || !scriptEl) {
            return;
        }

        var urls;
        try {
            urls = JSON.parse(scriptEl.textContent);
        } catch (e) {
            urls = [];
        }
        if (!urls || !urls.length) {
            return;
        }

        var stage = container.querySelector('.gallery-slideshow__stage');
        if (!stage) {
            stage = document.createElement('div');
            stage.className = 'gallery-slideshow__stage';
            stage.setAttribute('aria-live', 'polite');
            container.appendChild(stage);
        }

        stage.style.setProperty('--gallery-fade-ms', fadeMs + 'ms');

        var layerA = document.createElement('img');
        var layerB = document.createElement('img');
        layerA.className = 'gallery-slideshow__layer gallery-slideshow__layer--active';
        layerB.className = 'gallery-slideshow__layer';
        layerA.alt = '';
        layerB.alt = '';
        stage.appendChild(layerA);
        stage.appendChild(layerB);

        var index = 0;
        var activeLayer = layerA;
        var inactiveLayer = layerB;
        var isTransitioning = false;
        var paused = false;
        var timerId = null;

        function wrapIndex(i) {
            var n = urls.length;
            return ((i % n) + n) % n;
        }

        function swapLayers() {
            var tmp = activeLayer;
            activeLayer = inactiveLayer;
            inactiveLayer = tmp;
        }

        function showLayer(layer) {
            layer.classList.add('gallery-slideshow__layer--active');
        }

        function hideLayer(layer) {
            layer.classList.remove('gallery-slideshow__layer--active');
        }

        function scheduleAutoplay() {
            if (timerId) {
                clearInterval(timerId);
                timerId = null;
            }
            if (urls.length < 2) {
                return;
            }
            timerId = setInterval(function () {
                if (!paused && !isTransitioning) {
                    goTo(index + 1);
                }
            }, interval);
        }

        function finishTransition(targetIndex) {
            activeLayer.removeAttribute('src');
            swapLayers();
            index = targetIndex;
            isTransitioning = false;
        }

        function goTo(targetIndex) {
            if (isTransitioning || urls.length === 0) {
                return;
            }
            targetIndex = wrapIndex(targetIndex);
            if (targetIndex === index && activeLayer.src) {
                return;
            }

            isTransitioning = true;
            var url = urls[targetIndex];

            preload(url).then(function () {
                inactiveLayer.src = url;
                hideLayer(activeLayer);
                showLayer(inactiveLayer);

                var done = false;
                function onEnd(e) {
                    if (done || (e && e.propertyName !== 'opacity')) {
                        return;
                    }
                    done = true;
                    inactiveLayer.removeEventListener('transitionend', onEnd);
                    clearTimeout(fallbackId);
                    finishTransition(targetIndex);
                }

                var fallbackId = setTimeout(function () {
                    onEnd(null);
                }, fadeMs + 100);

                inactiveLayer.addEventListener('transitionend', onEnd);
            }).catch(function () {
                isTransitioning = false;
            });
        }

        preload(urls[0]).then(function () {
            activeLayer.src = urls[0];
            showLayer(activeLayer);
            scheduleAutoplay();
        });

        if (pauseOnHover) {
            container.addEventListener('mouseenter', function () {
                paused = true;
            });
            container.addEventListener('mouseleave', function () {
                paused = false;
            });
        }

        if (arrowKeys) {
            document.addEventListener('keydown', function (e) {
                if (e.keyCode === 37) {
                    e.preventDefault();
                    goTo(index - 1);
                } else if (e.keyCode === 39) {
                    e.preventDefault();
                    goTo(index + 1);
                }
            });
        }

        return {
            goTo: goTo,
            getIndex: function () { return index; }
        };
    }

    return { init: init };
})();
