import * as multipartParser from './multipart-parser.js';

document.getElementById('processForm').addEventListener('submit', function (event) {
    event.preventDefault();

    const fileInput = document.getElementById('file');
    const action = document.getElementById('action').value;
    const name = document.getElementById('name').value;

    if (fileInput.files.length === 0) {
        alert('Please select a file.');
        return;
    }
    const file = fileInput.files[0];
    const formData = new FormData();
    formData.append("file", file);
    formData.append('action', action);
    if (name) {
        formData.append('name', name);
    }

    const imageResult = document.getElementById('imageResult');
    imageResult.textContent = "Processing image...";

    fetch('http://localhost:8080/process', {
        method: 'POST',
        body: formData
    })
        .then(response => {
            if (!response.ok) {
                throw new Error('Failed to process image');
            }
            return response.blob().then(blob => ({ blob, response }));
        })
        .then(({ blob, response }) => {
            console.log('Response Headers:', ...response.headers)
            const reader = new FileReader();
            reader.onload = function (event) {
                const multipartData = event.target.result;
                const boundary = response.headers.get('Content-Type').split('boundary=')[1];
                const parts = multipartParser.Parse(new Uint8Array(event.target.result), boundary);
                
                console.log('Blob size:', blob.size);
                console.log('parts:', parts);
                parts.forEach(part => {
                    if (part.type === 'application/json') {
                        const jsonData = JSON.parse(new TextDecoder().decode(part.data));
                        console.log('JSON data:', jsonData);
                        imageResult.textContent = jsonData.message;
                    } else if (part.type.startsWith('image/')) {
                        const imageBlob = new Blob([part.data], { type: part.type });
                        const imageUrl = URL.createObjectURL(imageBlob);
                        const imagePreview = document.getElementById('imagePreview');
                        imagePreview.src = imageUrl;
                        imagePreview.style.display = 'block';
                    }
                });
            };
            reader.readAsArrayBuffer(blob);
        })
        .catch(error => {
            console.error('Fetch error:', error);
            alert('Failed to process image');
        });
});

document.getElementById('file').addEventListener('change', function (event) {
    const fileInput = event.target;
    const file = fileInput.files[0];
    const imagePreview = document.getElementById('imagePreview');

    if (file) {
        const reader = new FileReader();
        reader.onload = function (e) {
            imagePreview.src = e.target.result;
            imagePreview.style.display = 'block';
        };
        reader.readAsDataURL(file);
    } else {
        imagePreview.src = '';
        imagePreview.style.display = 'none';
    }
});