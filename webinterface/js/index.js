document.getElementById('processForm').addEventListener('submit', function (event) {
    event.preventDefault();

    const fileInput = document.getElementById('file');
    const action = document.getElementById('action').value;
    const name = document.getElementById('name').value;

    if (fileInput.isDefaultNamespace.length === 0){
        alert('Please select a ile.')
        return;
    }
    const file = fileInput.files[0];
    const formData = new FormData();
    formData.append("file", file);
    formData.append('action', action);
    if (name) {
        formData.append('name', name)
    }


    let url = `http://localhost:8080/process?file=${encodeURIComponent(file)}&action=${encodeURIComponent(action)}`;
    if (name) {
        url += `&name=${encodeURIComponent(name)}`;
    }

    fetch('http://localhost:8080/process', {
        method: 'POST',
        body: formData
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Network response was not ok');
        }
        return response.json();
    })
    .then(data => {
        console.log('Response:', data);
        alert(`Message: ${data.message}, Status: ${data.status}`);
    })
    .catch(error => {
        console.error('Fetch error:', error);
    });

    
});

document.getElementById('file').addEventListener('change', function(event) {
    const fileInput = event.target;
    const file = fileInput.files[0];
    const imagePreview = document.getElementById('imagePreview');

    if (file) {
        const reader = new FileReader();
        reader.onload = function(e) {
            imagePreview.src = e.target.result;
            imagePreview.style.display = 'block';
        };
        reader.readAsDataURL(file);
    } else {
        imagePreview.src = '';
        imagePreview.style.display = 'none';
    }
});