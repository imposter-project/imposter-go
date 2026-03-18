/**
 * WSDL Web - A lightweight WSDL/SOAP service viewer.
 */
var WSDLWeb = (function() {
  'use strict';

  var contentEl;
  var currentWSDL = null;

  function init(configs) {
    contentEl = document.getElementById('wsdl-content');

    if (!configs || configs.length === 0) {
      contentEl.innerHTML = '<p class="placeholder">No WSDL files found. Add a SOAP plugin configuration with a wsdlFile to get started.</p>';
      return;
    }

    // Build selector if multiple WSDLs
    var selectorEl = document.getElementById('wsdl-selector');
    if (configs.length === 1) {
      loadWSDL(configs[0]);
    } else {
      var select = document.createElement('select');
      select.id = 'wsdl-select';

      var defaultOpt = document.createElement('option');
      defaultOpt.value = '';
      defaultOpt.textContent = 'Select a WSDL...';
      select.appendChild(defaultOpt);

      for (var i = 0; i < configs.length; i++) {
        var opt = document.createElement('option');
        opt.value = i;
        opt.textContent = configs[i].name;
        select.appendChild(opt);
      }

      select.addEventListener('change', function() {
        if (select.value !== '') {
          loadWSDL(configs[parseInt(select.value)]);
        }
      });

      selectorEl.appendChild(select);

      // Auto-load first WSDL
      loadWSDL(configs[0]);
      select.value = '0';
    }
  }

  function loadWSDL(config) {
    contentEl.innerHTML = '<p class="loading">Loading WSDL...</p>';

    var xhr = new XMLHttpRequest();
    xhr.open('GET', config.url, true);
    xhr.onreadystatechange = function() {
      if (xhr.readyState !== 4) return;
      if (xhr.status === 200) {
        try {
          var parser = new DOMParser();
          var xmlDoc = parser.parseFromString(xhr.responseText, 'application/xml');

          var parseError = xmlDoc.querySelector('parsererror');
          if (parseError) {
            contentEl.innerHTML = '<div class="error">Failed to parse WSDL XML: ' + escapeHtml(parseError.textContent) + '</div>';
            return;
          }

          currentWSDL = xmlDoc;
          renderWSDL(xmlDoc, xhr.responseText, config.name);
        } catch (e) {
          contentEl.innerHTML = '<div class="error">Error processing WSDL: ' + escapeHtml(e.message) + '</div>';
        }
      } else {
        contentEl.innerHTML = '<div class="error">Failed to load WSDL: HTTP ' + xhr.status + '</div>';
      }
    };
    xhr.send();
  }

  function renderWSDL(xmlDoc, rawXml, name) {
    var html = '';
    var root = xmlDoc.documentElement;

    // Detect WSDL version and namespace
    var isWSDL2 = root.localName === 'description';
    var targetNamespace = root.getAttribute('targetNamespace') || '';

    // Info section
    html += '<div class="wsdl-info">';
    html += '<span class="info-label">WSDL: </span><span class="info-value">' + escapeHtml(name) + '</span>';
    if (targetNamespace) {
      html += '<br><span class="info-label">Target Namespace: </span><span class="info-value">' + escapeHtml(targetNamespace) + '</span>';
    }
    html += '</div>';

    if (isWSDL2) {
      html += renderWSDL2(root);
    } else {
      html += renderWSDL1(root);
    }

    // Raw XML section
    html += '<div class="wsdl-raw">';
    html += '<div class="wsdl-raw-header" onclick="WSDLWeb.toggleSection(this)">Raw WSDL <span class="toggle">Show</span></div>';
    html += '<div class="wsdl-raw-body"><pre>' + escapeHtml(rawXml) + '</pre></div>';
    html += '</div>';

    contentEl.innerHTML = html;
  }

  function renderWSDL1(root) {
    var html = '';
    var services = getLocalElements(root, 'service');
    var messages = getLocalElements(root, 'message');
    var portTypes = getLocalElements(root, 'portType');
    var bindings = getLocalElements(root, 'binding');

    // Build a lookup of messages by name
    var messageMap = {};
    for (var i = 0; i < messages.length; i++) {
      var msgName = messages[i].getAttribute('name');
      var parts = getLocalElements(messages[i], 'part');
      var partList = [];
      for (var j = 0; j < parts.length; j++) {
        partList.push({
          name: parts[j].getAttribute('name') || '',
          element: parts[j].getAttribute('element') || '',
          type: parts[j].getAttribute('type') || ''
        });
      }
      messageMap[msgName] = partList;
    }

    // Build a lookup of portType operations
    var portTypeMap = {};
    for (var i = 0; i < portTypes.length; i++) {
      var ptName = portTypes[i].getAttribute('name');
      var ops = getLocalElements(portTypes[i], 'operation');
      var opList = [];
      for (var j = 0; j < ops.length; j++) {
        var input = getLocalElement(ops[j], 'input');
        var output = getLocalElement(ops[j], 'output');
        var faults = getLocalElements(ops[j], 'fault');
        opList.push({
          name: ops[j].getAttribute('name') || '',
          inputMessage: input ? stripPrefix(input.getAttribute('message') || '') : '',
          outputMessage: output ? stripPrefix(output.getAttribute('message') || '') : '',
          faults: faults.map(function(f) {
            return { name: f.getAttribute('name') || '', message: stripPrefix(f.getAttribute('message') || '') };
          })
        });
      }
      portTypeMap[ptName] = opList;
    }

    // Build binding lookup (binding -> portType, operations with soapAction)
    var bindingMap = {};
    for (var i = 0; i < bindings.length; i++) {
      var bName = bindings[i].getAttribute('name');
      var bType = stripPrefix(bindings[i].getAttribute('type') || '');
      var bOps = getLocalElements(bindings[i], 'operation');
      var bOpMap = {};
      for (var j = 0; j < bOps.length; j++) {
        var opName = bOps[j].getAttribute('name');
        var soapOp = getSoapElement(bOps[j], 'operation');
        bOpMap[opName] = {
          soapAction: soapOp ? (soapOp.getAttribute('soapAction') || '') : ''
        };
      }
      bindingMap[bName] = { portType: bType, operations: bOpMap };
    }

    // Render services
    for (var i = 0; i < services.length; i++) {
      var svcName = services[i].getAttribute('name') || 'Service';
      var ports = getLocalElements(services[i], 'port');

      html += '<div class="wsdl-service">';
      html += '<div class="wsdl-service-header" onclick="WSDLWeb.toggleSection(this)">' + escapeHtml(svcName) + ' <span class="toggle">Collapse</span></div>';
      html += '<div class="wsdl-service-body open">';

      for (var j = 0; j < ports.length; j++) {
        var portName = ports[j].getAttribute('name') || '';
        var portBinding = stripPrefix(ports[j].getAttribute('binding') || '');
        var addressEl = getSoapElement(ports[j], 'address') || getElementByLocalName(ports[j], 'address');
        var address = addressEl ? (addressEl.getAttribute('location') || '') : '';

        html += '<div class="wsdl-port">';
        html += '<div class="wsdl-port-header">';
        html += '<span>' + escapeHtml(portName) + '</span>';
        html += '<span class="binding-info">Binding: ' + escapeHtml(portBinding) + '</span>';
        if (address) {
          html += '<span class="address">' + escapeHtml(address) + '</span>';
        }
        html += '</div>';

        // Get operations from the binding's portType
        var binding = bindingMap[portBinding];
        var operations = binding ? (portTypeMap[binding.portType] || []) : [];

        for (var k = 0; k < operations.length; k++) {
          var op = operations[k];
          var bindingOp = binding ? (binding.operations[op.name] || {}) : {};
          var opId = 'op-' + i + '-' + j + '-' + k;

          html += '<div class="wsdl-operation">';
          html += '<div class="wsdl-operation-header" onclick="WSDLWeb.toggleDetail(\'' + opId + '\')">';
          html += '<span class="method-badge">SOAP</span>';
          html += '<span class="operation-name">' + escapeHtml(op.name) + '</span>';
          if (bindingOp.soapAction) {
            html += '<span class="soap-action">SOAPAction: ' + escapeHtml(bindingOp.soapAction) + '</span>';
          }
          html += '</div>';

          html += '<div class="wsdl-operation-detail" id="' + opId + '">';

          // Input message
          if (op.inputMessage) {
            html += '<h4>Input</h4>';
            html += renderMessage(op.inputMessage, messageMap);
          }

          // Output message
          if (op.outputMessage) {
            html += '<h4>Output</h4>';
            html += renderMessage(op.outputMessage, messageMap);
          }

          // Faults
          if (op.faults.length > 0) {
            html += '<h4>Faults</h4>';
            for (var f = 0; f < op.faults.length; f++) {
              html += renderMessage(op.faults[f].message, messageMap, op.faults[f].name);
            }
          }

          html += '</div>'; // operation-detail
          html += '</div>'; // operation
        }

        html += '</div>'; // port
      }

      html += '</div>'; // service-body
      html += '</div>'; // service
    }

    // If no services found, show portTypes directly
    if (services.length === 0 && portTypes.length > 0) {
      for (var i = 0; i < portTypes.length; i++) {
        var ptName = portTypes[i].getAttribute('name') || 'PortType';
        var ops = portTypeMap[ptName] || [];

        html += '<div class="wsdl-service">';
        html += '<div class="wsdl-service-header" onclick="WSDLWeb.toggleSection(this)">' + escapeHtml(ptName) + ' <span class="toggle">Collapse</span></div>';
        html += '<div class="wsdl-service-body open">';

        for (var k = 0; k < ops.length; k++) {
          var op = ops[k];
          var opId = 'pt-op-' + i + '-' + k;

          html += '<div class="wsdl-operation">';
          html += '<div class="wsdl-operation-header" onclick="WSDLWeb.toggleDetail(\'' + opId + '\')">';
          html += '<span class="method-badge">SOAP</span>';
          html += '<span class="operation-name">' + escapeHtml(op.name) + '</span>';
          html += '</div>';

          html += '<div class="wsdl-operation-detail" id="' + opId + '">';
          if (op.inputMessage) {
            html += '<h4>Input</h4>';
            html += renderMessage(op.inputMessage, messageMap);
          }
          if (op.outputMessage) {
            html += '<h4>Output</h4>';
            html += renderMessage(op.outputMessage, messageMap);
          }
          html += '</div>';
          html += '</div>';
        }

        html += '</div>';
        html += '</div>';
      }
    }

    return html;
  }

  function renderWSDL2(root) {
    var html = '';
    var services = getLocalElements(root, 'service');
    var interfaces = getLocalElements(root, 'interface');

    // Build interface lookup
    var interfaceMap = {};
    for (var i = 0; i < interfaces.length; i++) {
      var ifName = interfaces[i].getAttribute('name');
      var ops = getLocalElements(interfaces[i], 'operation');
      var opList = [];
      for (var j = 0; j < ops.length; j++) {
        var input = getLocalElement(ops[j], 'input');
        var output = getLocalElement(ops[j], 'output');
        opList.push({
          name: ops[j].getAttribute('name') || '',
          pattern: ops[j].getAttribute('pattern') || '',
          inputElement: input ? (input.getAttribute('element') || '') : '',
          outputElement: output ? (output.getAttribute('element') || '') : ''
        });
      }
      interfaceMap[ifName] = opList;
    }

    for (var i = 0; i < services.length; i++) {
      var svcName = services[i].getAttribute('name') || 'Service';
      var svcInterface = stripPrefix(services[i].getAttribute('interface') || '');
      var endpoints = getLocalElements(services[i], 'endpoint');

      html += '<div class="wsdl-service">';
      html += '<div class="wsdl-service-header" onclick="WSDLWeb.toggleSection(this)">' + escapeHtml(svcName) + ' <span class="toggle">Collapse</span></div>';
      html += '<div class="wsdl-service-body open">';

      for (var j = 0; j < endpoints.length; j++) {
        var epName = endpoints[j].getAttribute('name') || '';
        var address = endpoints[j].getAttribute('address') || '';

        html += '<div class="wsdl-port">';
        html += '<div class="wsdl-port-header">';
        html += '<span>' + escapeHtml(epName) + '</span>';
        html += '<span class="binding-info">Interface: ' + escapeHtml(svcInterface) + '</span>';
        if (address) {
          html += '<span class="address">' + escapeHtml(address) + '</span>';
        }
        html += '</div>';

        var operations = interfaceMap[svcInterface] || [];
        for (var k = 0; k < operations.length; k++) {
          var op = operations[k];
          var opId = 'w2-op-' + i + '-' + j + '-' + k;

          html += '<div class="wsdl-operation">';
          html += '<div class="wsdl-operation-header" onclick="WSDLWeb.toggleDetail(\'' + opId + '\')">';
          html += '<span class="method-badge">SOAP</span>';
          html += '<span class="operation-name">' + escapeHtml(op.name) + '</span>';
          html += '</div>';

          html += '<div class="wsdl-operation-detail" id="' + opId + '">';
          if (op.inputElement) {
            html += '<h4>Input</h4>';
            html += '<div class="message-info"><span class="msg-name">' + escapeHtml(op.inputElement) + '</span></div>';
          }
          if (op.outputElement) {
            html += '<h4>Output</h4>';
            html += '<div class="message-info"><span class="msg-name">' + escapeHtml(op.outputElement) + '</span></div>';
          }
          html += '</div>';
          html += '</div>';
        }

        html += '</div>';
      }

      html += '</div>';
      html += '</div>';
    }

    return html;
  }

  function renderMessage(msgName, messageMap, faultName) {
    var parts = messageMap[msgName];
    var html = '<div class="message-info">';
    html += '<span class="msg-name">' + escapeHtml(faultName || msgName) + '</span>';
    if (parts && parts.length > 0) {
      html += '<div class="msg-parts">';
      for (var i = 0; i < parts.length; i++) {
        html += '<div class="part">';
        html += '<span class="part-name">' + escapeHtml(parts[i].name) + '</span>: ';
        html += '<span class="part-type">' + escapeHtml(parts[i].element || parts[i].type || 'any') + '</span>';
        html += '</div>';
      }
      html += '</div>';
    }
    html += '</div>';
    return html;
  }

  // Helper: get child elements with a given local name (any namespace)
  function getLocalElements(parent, localName) {
    var result = [];
    var children = parent.childNodes;
    for (var i = 0; i < children.length; i++) {
      if (children[i].nodeType === 1 && children[i].localName === localName) {
        result.push(children[i]);
      }
    }
    return result;
  }

  function getLocalElement(parent, localName) {
    var elements = getLocalElements(parent, localName);
    return elements.length > 0 ? elements[0] : null;
  }

  // Helper: get SOAP-namespace elements (soap11 or soap12)
  function getSoapElement(parent, localName) {
    var soap11 = 'http://schemas.xmlsoap.org/wsdl/soap/';
    var soap12 = 'http://schemas.xmlsoap.org/wsdl/soap12/';
    var http = 'http://schemas.xmlsoap.org/wsdl/http/';

    var el = parent.getElementsByTagNameNS(soap11, localName);
    if (el.length > 0) return el[0];
    el = parent.getElementsByTagNameNS(soap12, localName);
    if (el.length > 0) return el[0];
    el = parent.getElementsByTagNameNS(http, localName);
    if (el.length > 0) return el[0];
    return null;
  }

  function getElementByLocalName(parent, localName) {
    var children = parent.childNodes;
    for (var i = 0; i < children.length; i++) {
      if (children[i].nodeType === 1 && children[i].localName === localName) {
        return children[i];
      }
    }
    return null;
  }

  function stripPrefix(qname) {
    var idx = qname.indexOf(':');
    return idx >= 0 ? qname.substring(idx + 1) : qname;
  }

  function escapeHtml(text) {
    var div = document.createElement('div');
    div.appendChild(document.createTextNode(text));
    return div.innerHTML;
  }

  function toggleSection(header) {
    var body = header.nextElementSibling;
    var toggle = header.querySelector('.toggle');
    if (body.classList.contains('open')) {
      body.classList.remove('open');
      if (toggle) toggle.textContent = 'Show';
    } else {
      body.classList.add('open');
      if (toggle) toggle.textContent = 'Collapse';
    }
  }

  function toggleDetail(id) {
    var el = document.getElementById(id);
    if (el) {
      el.classList.toggle('open');
    }
  }

  return {
    init: init,
    toggleSection: toggleSection,
    toggleDetail: toggleDetail
  };
})();
