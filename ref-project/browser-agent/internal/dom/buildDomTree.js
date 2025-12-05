(args => {
    const doHighlightElements = args.doHighlightElements || false;
    const focusHighlightIndex = args.focusHighlightIndex || -1;
    const viewportExpansion = args.viewportExpansion || 0;
    const debugMode = args.debugMode || false;

    const perfStart = performance.now();
    const perfMetrics = {};

    function isVisible(element) {
        if (!element) return false;
        const style = window.getComputedStyle(element);
        if (style.display === 'none' || style.visibility === 'hidden' || style.opacity === '0') {
            return false;
        }
        const rect = element.getBoundingClientRect();
        return rect.width > 0 && rect.height > 0;
    }

    function isInteractive(element) {
        const interactiveTags = ['a', 'button', 'input', 'select', 'textarea', 'details', 'summary'];
        const interactiveRoles = ['button', 'link', 'checkbox', 'radio', 'tab', 'menuitem'];

        if (interactiveTags.includes(element.tagName.toLowerCase())) return true;
        if (element.hasAttribute('onclick')) return true;
        if (element.hasAttribute('role') && interactiveRoles.includes(element.getAttribute('role'))) return true;
        if (element.getAttribute('contenteditable') === 'true') return true;
        if (element.tabIndex >= 0) return true;

        return false;
    }

    function isInViewport(element, expansion = 0) {
        const rect = element.getBoundingClientRect();
        return (
            rect.top >= -expansion &&
            rect.left >= -expansion &&
            rect.bottom <= (window.innerHeight || document.documentElement.clientHeight) + expansion &&
            rect.right <= (window.innerWidth || document.documentElement.clientWidth) + expansion
        );
    }

    function getXPath(element) {
        if (element.id !== '') {
            return `//*[@id="${element.id}"]`;
        }
        if (element === document.body) {
            return '/html/body';
        }

        let ix = 0;
        const siblings = element.parentNode ? element.parentNode.childNodes : [];
        for (let i = 0; i < siblings.length; i++) {
            const sibling = siblings[i];
            if (sibling === element) {
                const parentPath = element.parentNode ? getXPath(element.parentNode) : '';
                return `${parentPath}/${element.tagName.toLowerCase()}[${ix + 1}]`;
            }
            if (sibling.nodeType === 1 && sibling.tagName === element.tagName) {
                ix++;
            }
        }
        return '';
    }

    function getAttributes(element) {
        const attrs = {};
        for (let i = 0; i < element.attributes.length; i++) {
            const attr = element.attributes[i];
            attrs[attr.name] = attr.value;
        }
        return attrs;
    }

    let nodeId = 0;
    let highlightIndex = 0;
    const nodeMap = {};

    function processNode(node, parentId = null) {
        if (node.nodeType === Node.TEXT_NODE) {
            const text = node.textContent.trim();
            if (text.length === 0) return null;

            const id = String(nodeId++);
            nodeMap[id] = {
                type: 'TEXT_NODE',
                text: text,
                isVisible: true
            };
            return id;
        }

        if (node.nodeType !== Node.ELEMENT_NODE) return null;

        const element = node;
        const visible = isVisible(element);
        const interactive = isInteractive(element);
        const inViewport = isInViewport(element, viewportExpansion);

        const id = String(nodeId++);
        const children = [];

        let currentHighlightIndex = null;
        if (interactive && visible && inViewport && doHighlightElements) {
            currentHighlightIndex = highlightIndex++;
        }

        for (let i = 0; i < node.childNodes.length; i++) {
            const childId = processNode(node.childNodes[i], id);
            if (childId !== null) {
                children.push(childId);
            }
        }

        const nodeData = {
            type: 'ELEMENT_NODE',
            tagName: element.tagName.toLowerCase(),
            xpath: getXPath(element),
            attributes: getAttributes(element),
            children: children,
            isVisible: visible,
            isInteractive: interactive,
            isTopElement: true,
            isInViewport: inViewport,
            shadowRoot: !!element.shadowRoot
        };

        if (currentHighlightIndex !== null) {
            nodeData.highlightIndex = currentHighlightIndex;
        }

        if (inViewport) {
            nodeData.viewport = {
                width: window.innerWidth,
                height: window.innerHeight
            };
        }

        nodeMap[id] = nodeData;
        return id;
    }

    const rootId = processNode(document.body);

    perfMetrics.totalTime = performance.now() - perfStart;
    perfMetrics.nodeCount = nodeId;
    perfMetrics.highlightCount = highlightIndex;

    return {
        map: nodeMap,
        rootId: rootId,
        perfMetrics: debugMode ? perfMetrics : null
    };
});