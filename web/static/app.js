$(document).ready(() => {
    $.fn.dataTable.ext.order['dom-order'] = function( _, col ) { 
        return this.api().column( col, {order:'index'} ).nodes().map(td => $(td).data('order'));
    };

    $('#events').DataTable({
        lengthMenu: [ [10, 25, 50, 100, -1], [10, 25, 50, 100, "All"] ],
        pageLength: 25,
        order: [[0, 'desc']],
        columnDefs: [
            { targets: 'time', orderDataType: 'dom-order' },
            { targets: 'start', orderDataType: 'dom-order' },
            { targets: 'end', orderDataType: 'dom-order' },
            { targets: 'duration', orderDataType: 'dom-order', type: 'numeric' }
        ],
        language: {
            emptyTable: "No events received yet - push something or comment on a PR, or make sure your webhooks are correctly setup.<br>See the <a href='/jobs'>Jobs</a> instead?"
        }
    });

    $('#jobs').DataTable({
        lengthMenu: [ [10, 25, 50, 100, -1], [10, 25, 50, 100, "All"] ],
        pageLength: 25,
        order: [[0, 'desc']],
        columnDefs: [
            { targets: 'time', orderDataType: 'dom-order' },
            { targets: 'start', orderDataType: 'dom-order' },
            { targets: 'end', orderDataType: 'dom-order' },
            { targets: 'duration', orderDataType: 'dom-order', type: 'numeric' }
        ]
    });

    $('#pools').DataTable({
        lengthMenu: [ [10, 25, 50, 100, -1], [10, 25, 50, 100, "All"] ],
        pageLength: 25,
        order: [[0, 'desc']],
        columnDefs: [
            { targets: 'updatedAt', orderDataType: 'dom-order' }
        ],
        language: {
            emptyTable: "Nothing in the Merge Pools at the moment.<br>See the <a href='/merge/history'>Merge History</a> instead?"
        }
    });

    $('#records').DataTable({
        lengthMenu: [ [10, 25, 50, 100, -1], [10, 25, 50, 100, "All"] ],
        pageLength: 25,
        order: [[0, 'desc']],
        columnDefs: [
            { targets: 'time', orderDataType: 'dom-order' }
        ],
        language: {
            emptyTable: "Nothing in the Merge History at the moment.<br>Approve Pull Requests and you will see them here once they have been merged."
        }
    });

});

(function(){
    const copyToClipboard = (text) => {
        navigator.permissions.query({name: "clipboard-write"}).then(result => {
            if (result.state == "granted" || result.state == "prompt") {
                navigator.clipboard.writeText(text).then(function() {
                    console.log("ok, copied to clipboard");
                }, function() {
                    console.log("oops, failed to copy to clipboard");
                });
            } else {
                console.log("no permission to write to the clipboard", result);
            }
        });
    };

    document.querySelectorAll('.event-copy-guid-to-clipboard').forEach(element => {
        element.addEventListener('click', event => {
            const eventGuid = event.target.dataset.guid;
            if (eventGuid) {
                copyToClipboard(eventGuid);
            }
        })
    });
})();