document.getElementById('processForm').addEventListener('submit', function (event) {
    event.preventDefault();

    const file = document.getElementById('file').value;
    const action = document.getElementById('action').value;
    const name = document.getElementById('name').value;

    let url = `http://localhost:8080/process?file=${encodeURIComponent(file)}&action=${encodeURIComponent(action)}`;
    if (name) {
        url += `&name=${encodeURIComponent(name)}`;
    }

    fetch(url)
        .then(response => {
            if (!response.ok) {
                throw new Error('Network response was not ok');
            }
            return response.text(); // Get raw text for closer inspection
        })
        .then(text => {
            console.log('Raw Response:', text); // Log raw response data
            try {
                const data = JSON.parse(text); // Only parse if content seems valid JSON
                console.log('Parsed JSON:', data);
                alert(`Message: ${data.message}, Status: ${data.status}`);
            } catch (error) {
                console.error('Failed to parse JSON:', error);
            }
        })
        .catch(error => {
            console.error('Fetch error:', error);
        });
});
