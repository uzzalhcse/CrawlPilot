export function useNavigationPrevention() {
  let handlers: Array<{ element: any; event: string; handler: (e: Event) => void }> = []

  const preventDefault = (e: Event) => {
    e.preventDefault()
    e.stopPropagation()
  }

  const initNavigationPrevention = () => {
    // Prevent all link clicks
    const linkHandler = (e: Event) => {
      const target = e.target as Element
      if (target.closest('a') && !target.closest('#crawlify-selector-overlay')) {
        preventDefault(e)
      }
    }

    // Prevent form submissions
    const formHandler = (e: Event) => {
      const target = e.target as Element
      if (target.closest('form') && !target.closest('#crawlify-selector-overlay')) {
        preventDefault(e)
      }
    }

    // Prevent navigation via keyboard
    const keyHandler = (e: KeyboardEvent) => {
      // Prevent F5, Ctrl+R, etc.
      if ((e.key === 'F5') || (e.ctrlKey && e.key === 'r')) {
        preventDefault(e)
      }
    }

    document.addEventListener('click', linkHandler, true)
    document.addEventListener('submit', formHandler, true)
    document.addEventListener('keydown', keyHandler, true)

    handlers = [
      { element: document, event: 'click', handler: linkHandler },
      { element: document, event: 'submit', handler: formHandler },
      { element: document, event: 'keydown', handler: keyHandler }
    ]
  }

  const cleanupNavigationPrevention = () => {
    handlers.forEach(({ element, event, handler }) => {
      element.removeEventListener(event, handler, true)
    })
    handlers = []
  }

  return {
    initNavigationPrevention,
    cleanupNavigationPrevention
  }
}
