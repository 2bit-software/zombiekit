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
