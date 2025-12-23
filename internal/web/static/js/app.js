// Brains web interface JavaScript

// Update sidebar active state when HTMX content is swapped
document.body.addEventListener('htmx:afterSwap', function(event) {
    // Only process content swaps
    if (event.detail.target.id !== 'content') {
        return;
    }

    // Get the current URL path
    const currentPath = window.location.pathname;

    // Update sidebar link active states
    document.querySelectorAll('.sidebar-link').forEach(function(link) {
        const linkPath = link.getAttribute('href');

        // Remove all active classes
        link.classList.remove('bg-gray-900', 'text-white');
        link.classList.add('text-gray-300');

        // Add active classes if this link matches current path
        if (linkPath === '/' && currentPath === '/') {
            link.classList.add('bg-gray-900', 'text-white');
            link.classList.remove('text-gray-300');
        } else if (linkPath !== '/' && currentPath.startsWith(linkPath)) {
            link.classList.add('bg-gray-900', 'text-white');
            link.classList.remove('text-gray-300');
        }
    });
});

// Handle browser back/forward navigation
window.addEventListener('popstate', function(event) {
    // HTMX handles the content update, we just need to update sidebar
    const currentPath = window.location.pathname;

    document.querySelectorAll('.sidebar-link').forEach(function(link) {
        const linkPath = link.getAttribute('href');

        link.classList.remove('bg-gray-900', 'text-white');
        link.classList.add('text-gray-300');

        if (linkPath === '/' && currentPath === '/') {
            link.classList.add('bg-gray-900', 'text-white');
            link.classList.remove('text-gray-300');
        } else if (linkPath !== '/' && currentPath.startsWith(linkPath)) {
            link.classList.add('bg-gray-900', 'text-white');
            link.classList.remove('text-gray-300');
        }
    });
});

// =====================================
// Search functionality
// =====================================

// Track currently selected search result index
let selectedResultIndex = -1;

// Clear search results and input after navigation
document.body.addEventListener('htmx:afterSwap', function(event) {
    // Clear search when main content is swapped (user navigated via search result)
    if (event.detail.target.id === 'content') {
        clearSearch();
    }
});

// Clear search helper function
function clearSearch() {
    const searchInput = document.getElementById('search-input');
    const searchResults = document.getElementById('search-results');
    if (searchInput) searchInput.value = '';
    if (searchResults) searchResults.innerHTML = '';
    selectedResultIndex = -1;
}

// Click outside handler - close dropdown when clicking outside search container
document.addEventListener('click', function(event) {
    const searchContainer = document.getElementById('search-container');
    if (searchContainer && !searchContainer.contains(event.target)) {
        const searchResults = document.getElementById('search-results');
        if (searchResults) searchResults.innerHTML = '';
        selectedResultIndex = -1;
    }
});

// Keyboard navigation for search results
document.addEventListener('keydown', function(event) {
    const searchInput = document.getElementById('search-input');
    const searchResults = document.getElementById('search-results');

    // "/" key focuses search bar (unless already in an input/textarea)
    if (event.key === '/' && document.activeElement.tagName !== 'INPUT' && document.activeElement.tagName !== 'TEXTAREA') {
        event.preventDefault();
        if (searchInput) searchInput.focus();
        return;
    }

    // Only handle other keys when search input is focused
    if (document.activeElement !== searchInput) return;

    const results = searchResults ? searchResults.querySelectorAll('[data-search-result]') : [];
    if (results.length === 0) return;

    switch (event.key) {
        case 'ArrowDown':
            event.preventDefault();
            selectedResultIndex = Math.min(selectedResultIndex + 1, results.length - 1);
            updateResultSelection(results);
            break;

        case 'ArrowUp':
            event.preventDefault();
            selectedResultIndex = Math.max(selectedResultIndex - 1, -1);
            updateResultSelection(results);
            break;

        case 'Enter':
            if (selectedResultIndex >= 0 && selectedResultIndex < results.length) {
                event.preventDefault();
                // Trigger HTMX navigation on selected result
                htmx.trigger(results[selectedResultIndex], 'click');
            }
            break;

        case 'Escape':
            event.preventDefault();
            clearSearch();
            searchInput.blur();
            break;
    }
});

// Update visual selection of search results
function updateResultSelection(results) {
    results.forEach(function(result, index) {
        if (index === selectedResultIndex) {
            result.classList.add('bg-blue-50', 'text-blue-700');
            result.classList.remove('text-gray-700');
            result.scrollIntoView({ block: 'nearest' });
        } else {
            result.classList.remove('bg-blue-50', 'text-blue-700');
            result.classList.add('text-gray-700');
        }
    });
}

// Reset selection when new search results are loaded
document.body.addEventListener('htmx:afterSwap', function(event) {
    if (event.detail.target.id === 'search-results') {
        selectedResultIndex = -1;
    }
});
