from setuptools import setup, find_packages

setup(
    name="ml-core",
    version="0.1.0",
    packages=find_packages(),
    install_requires=["torch", "pandas", "scikit-learn"],
)
