        document.getElementById('processForm').addEventListener('submit', function(event) {
            event.preventDefault();

            const file = document.getElementById('file').value;
            const action = document.getElementById('action').value;
            const name = document.getElementById('name').value;

 let url = `http://localhost:8080/process?file=${encodeURIComponent(file)}&action=${encodeURIComponent(action)}`;
    if (name) {
        url += `&name=${encodeURIComponent(name)}`;
    }

            fetch(url)
                .then(response => response.text())
                .then(data => {
                    alert(`Response: ${data}`);
                })
                .catch(error => {
                    console.error('Error:', error);
                    alert('An error occurred. Please try again.');
                });
        });

