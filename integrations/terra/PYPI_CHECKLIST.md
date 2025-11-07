# PyPI Submission Checklist

Complete checklist for publishing GenomeVedic to PyPI.

---

## Pre-submission Checklist

### Package Files

- [x] `setup.py` - Package configuration
- [x] `pyproject.toml` - Modern build configuration
- [x] `README.md` - Package description (used on PyPI)
- [x] `LICENSE` - MIT license file
- [x] `MANIFEST.in` - Include/exclude rules
- [x] `requirements.txt` - Core dependencies
- [x] `requirements-full.txt` - Full dependencies

### Code Quality

- [x] All modules have docstrings
- [x] Functions have type hints where appropriate
- [x] Code follows PEP 8 style guidelines
- [x] No hardcoded credentials or secrets
- [x] Error handling implemented
- [x] Logging configured appropriately

### Documentation

- [x] README.md with installation instructions
- [x] Quick start examples
- [x] API reference
- [x] Configuration guide
- [x] Troubleshooting section
- [x] LICENSE file
- [x] Detailed docs in `docs/TERRA_INTEGRATION.md`

### Testing

- [ ] Unit tests written (pytest)
- [ ] Integration tests created
- [ ] Test coverage > 80%
- [ ] Tests pass on Python 3.8+
- [ ] Manual testing completed

### Examples

- [x] Example Jupyter notebook created
- [x] Terra.bio quickstart example
- [x] Code snippets in README
- [x] Advanced usage examples

### Versioning

- [x] Version set in `__init__.py`
- [x] Version format: MAJOR.MINOR.PATCH (1.0.0)
- [x] CHANGELOG.md prepared (if applicable)

---

## Build Process

### Step 1: Verify Package Structure

```bash
cd /home/user/genomevedic/integrations/terra

# Check structure
tree -L 3
```

Expected structure:
```
.
├── setup.py
├── pyproject.toml
├── README.md
├── LICENSE
├── MANIFEST.in
├── requirements.txt
├── genomevedic_python/
│   ├── __init__.py
│   ├── api_client.py
│   ├── gcs_client.py
│   └── jupyter_widget.py
├── examples/
│   └── terra_quickstart.ipynb
└── docs/
    └── TERRA_INTEGRATION.md
```

### Step 2: Clean Build Artifacts

```bash
# Remove old builds
rm -rf build/ dist/ *.egg-info/

# Remove Python cache
find . -type d -name __pycache__ -exec rm -rf {} +
find . -type f -name "*.pyc" -delete
```

### Step 3: Install Build Tools

```bash
pip install --upgrade build twine setuptools wheel
```

### Step 4: Build Package

```bash
# Build source distribution and wheel
python -m build

# Expected output:
# dist/genomevedic-1.0.0.tar.gz
# dist/genomevedic-1.0.0-py3-none-any.whl
```

### Step 5: Verify Build

```bash
# Check package contents
tar -tzf dist/genomevedic-1.0.0.tar.gz | head -20
unzip -l dist/genomevedic-1.0.0-py3-none-any.whl

# Verify metadata
twine check dist/*
```

Expected output: `Checking dist/*... PASSED`

### Step 6: Test Installation Locally

```bash
# Create test environment
python -m venv test_env
source test_env/bin/activate

# Install from wheel
pip install dist/genomevedic-1.0.0-py3-none-any.whl

# Test import
python -c "import genomevedic; print(genomevedic.__version__)"

# Deactivate and remove
deactivate
rm -rf test_env
```

---

## PyPI Registration

### Step 1: Create PyPI Account

1. Go to https://pypi.org/account/register/
2. Create account
3. Verify email
4. Enable 2FA (recommended)

### Step 2: Create API Token

1. Go to https://pypi.org/manage/account/
2. Scroll to "API tokens"
3. Click "Add API token"
4. Name: "genomevedic-upload"
5. Scope: "Entire account" (for first upload)
6. Copy token (starts with `pypi-`)

### Step 3: Configure Credentials

Create `~/.pypirc`:

```ini
[distutils]
index-servers =
    pypi
    testpypi

[pypi]
username = __token__
password = pypi-AgEIcHlwaS5vcmc...

[testpypi]
repository = https://test.pypi.org/legacy/
username = __token__
password = pypi-AgENdGVzdC5weXBp...
```

Set permissions:
```bash
chmod 600 ~/.pypirc
```

---

## Test Upload (TestPyPI)

### Step 1: Upload to TestPyPI

```bash
twine upload --repository testpypi dist/*
```

### Step 2: Test Installation from TestPyPI

```bash
# Install from TestPyPI
pip install --index-url https://test.pypi.org/simple/ \
    --extra-index-url https://pypi.org/simple/ \
    genomevedic

# Test it works
python -c "import genomevedic; print(genomevedic.__version__)"
```

### Step 3: Verify on TestPyPI

Visit: https://test.pypi.org/project/genomevedic/

Check:
- [ ] README renders correctly
- [ ] Links work
- [ ] Classifiers correct
- [ ] Version correct
- [ ] Dependencies listed

---

## Production Upload (PyPI)

### Step 1: Final Checks

- [ ] All tests pass
- [ ] TestPyPI upload successful
- [ ] Documentation complete
- [ ] Version number finalized
- [ ] CHANGELOG updated

### Step 2: Upload to PyPI

```bash
# Upload to production PyPI
twine upload dist/*
```

### Step 3: Verify Upload

Visit: https://pypi.org/project/genomevedic/

### Step 4: Test Installation

```bash
# Install from PyPI
pip install genomevedic

# Test
python -c "import genomevedic as gv; gv.help()"
```

### Step 5: Tag Release

```bash
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
```

---

## Post-Publication

### Documentation

- [ ] Update GitHub README with PyPI badge
- [ ] Update installation instructions
- [ ] Announce on Terra.bio forum
- [ ] Create blog post (optional)

### Monitoring

- [ ] Watch PyPI download stats
- [ ] Monitor GitHub issues
- [ ] Track user feedback

### Maintenance

- [ ] Set up CI/CD for automated testing
- [ ] Plan next version features
- [ ] Document known issues

---

## Version Updates

### For Minor Updates (1.0.1, 1.0.2)

1. Update version in `__init__.py`
2. Update CHANGELOG
3. Build and test
4. Upload to PyPI
5. Tag release

### For Major Updates (1.1.0, 2.0.0)

1. Create release branch
2. Update version
3. Update documentation
4. Full test suite
5. Upload to TestPyPI first
6. Upload to PyPI
7. Announce changes

---

## Troubleshooting

### Issue: "File already exists on PyPI"

**Solution:** Increment version number. PyPI doesn't allow re-uploading same version.

```python
# In __init__.py
__version__ = "1.0.1"  # Increment
```

### Issue: "Invalid package structure"

**Solution:** Check MANIFEST.in and ensure all files included

```bash
python setup.py sdist
tar -tzf dist/genomevedic-*.tar.gz
```

### Issue: "Dependency resolution failed"

**Solution:** Check requirements versions

```bash
pip install -e .
pip check
```

### Issue: "README not rendering"

**Solution:** Ensure README.md is valid Markdown

```bash
# Install and use grip to preview
pip install grip
grip README.md
```

---

## Resources

- **PyPI Guide**: https://packaging.python.org/tutorials/packaging-projects/
- **Twine Docs**: https://twine.readthedocs.io/
- **PEP 517**: https://www.python.org/dev/peps/pep-0517/
- **setuptools**: https://setuptools.pypa.io/

---

## Automation (GitHub Actions)

Create `.github/workflows/publish.yml`:

```yaml
name: Publish to PyPI

on:
  release:
    types: [created]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2
    - name: Set up Python
      uses: actions/setup-python@v2
      with:
        python-version: '3.10'
    - name: Install dependencies
      run: |
        pip install build twine
    - name: Build package
      run: python -m build
    - name: Publish to PyPI
      env:
        TWINE_USERNAME: __token__
        TWINE_PASSWORD: ${{ secrets.PYPI_API_TOKEN }}
      run: twine upload dist/*
```

---

## Checklist Summary

**Pre-submission:** ✓ (All files created)
**Build Process:** Ready
**Testing:** Manual testing ready, automated tests optional
**PyPI Account:** User needs to create
**Upload:** Ready for execution

**Status: READY FOR PUBLICATION** 🚀

**Next Steps:**
1. Create PyPI account
2. Run build process
3. Test on TestPyPI
4. Upload to production PyPI
